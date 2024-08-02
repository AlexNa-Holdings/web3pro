package usb

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/google/gousb"
	"github.com/google/gousb/usbid"
	"github.com/rs/zerolog/log"
)

var ctx *gousb.Context

type USB_DEV struct {
	Type        string
	USB_ID      string
	Device      *gousb.Device
	Config      *gousb.Config
	Interface   *gousb.Interface
	EndpointOut *gousb.OutEndpoint
	EndpointIn  *gousb.InEndpoint
}

const USB_IDS = "usb.ids"

var usb_devices = []*USB_DEV{}
var usb_devices_mutex = &sync.Mutex{}

func Init() {
	init_usb_ids()
	go Loop()
}

func Loop() {
	ctx = gousb.NewContext()
	defer ctx.Close()

	ch := bus.Subscribe("usb")
	enum_ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case msg := <-ch:
			if msg.RespondTo != 0 {
				continue // ignore responses
			}
			go process(msg)

		case <-enum_ticker.C:
			enumerate()
		}
	}
}

func process(msg *bus.Message) {
	switch msg.Type {
	case "list": // list all devices
		list := bus.B_UsbList_Response{}

		ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {

			if desc.Path == nil || len(desc.Path) == 0 {
				return false // skip
			}

			conn := GetUSBDevice(GetUSB_ID(desc))
			connected := false
			if conn != nil {
				connected = conn.Device != nil
			}

			v, p := ResolveVendorProduct(uint16(desc.Vendor), uint16(desc.Product))
			list = append(list, bus.B_UsbList_Device{
				USB_ID:    GetUSB_ID(desc),
				Path:      GetPath(desc),
				Vendor:    v,
				VendorID:  uint16(desc.Vendor),
				Product:   p,
				ProductID: uint16(desc.Product),
				Connected: connected,
			})
			return false
		})

		msg.Respond(list, nil)
	case "write":
		req, ok := msg.Data.(*bus.B_UsbWrite)
		if !ok {
			log.Error().Msg("Invalid message data")
			msg.Respond(nil, bus.ErrInvalidMessageData)
			return
		}

		conn, err := OpenDevice(req.USB_ID)
		if err != nil {
			log.Error().Msg("Device not found")
			msg.Respond(nil, errors.New("device not found"))
			return
		}

		_, err = conn.EndpointOut.Write(req.Data)
		if err != nil {
			log.Error().Err(err).Msg("Error writing to device")
			msg.Respond(nil, err)
			return
		}
		msg.Respond(nil, nil)
	case "read":
		req, ok := msg.Data.(*bus.B_UsbRead)
		if !ok {
			log.Error().Msg("Invalid message data")
			msg.Respond(nil, bus.ErrInvalidMessageData)
			return
		}
		conn, err := OpenDevice(req.USB_ID)
		if err != nil {
			log.Error().Msg("Device not found")
			msg.Respond(nil, errors.New("device not found"))
			return
		}

		data := make([]byte, conn.EndpointIn.Desc.MaxPacketSize)
		n, err := conn.EndpointIn.Read(data)
		if err != nil {
			log.Error().Err(err).Msg("Error reading from device")
			msg.Respond(nil, err)
			return
		}

		msg.Respond(bus.B_UsbRead_Response{Data: data[:n]}, nil)
	}
}

func ResolveVendorProduct(vendor, product uint16) (string, string) {
	v := "Unknown"
	p := "Unknown"

	vendor_str := gousb.ID(vendor)
	product_str := gousb.ID(product)

	vendor_name, ok := usbid.Vendors[vendor_str]
	if ok {
		v = vendor_name.Name
		product_name, ok := vendor_name.Product[product_str]
		if ok {
			p = product_name.Name
		}
	}

	return v, p
}

func GetUSB_ID(d *gousb.DeviceDesc) string {
	return fmt.Sprintf("vid=%s,pid=%s,bus=%d,addr=%d", d.Vendor, d.Product, d.Bus, d.Address)
}

func GetPath(d *gousb.DeviceDesc) string {
	var sb strings.Builder
	for i, v := range d.Path {
		if i > 0 {
			sb.WriteString(":")
		}
		sb.WriteString(strconv.Itoa(v))
	}
	return sb.String()
}

func GetUSBDevice(id string) *USB_DEV {
	usb_devices_mutex.Lock()
	defer usb_devices_mutex.Unlock()

	for _, conn := range usb_devices {
		if conn.USB_ID == id {
			return conn
		}
	}
	return nil
}

func OpenDevice(id string) (*USB_DEV, error) {
	t := GetUSBDevice(id)
	if t == nil {
		return nil, errors.New("device not found")
	}

	if t.Device != nil {
		return t, nil // already opened
	}

	d, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		if desc.Path == nil || len(desc.Path) == 0 {
			return false // skip
		}

		if GetUSB_ID(desc) == id {
			return true
		}

		return false
	})

	if err != nil {
		return nil, err
	}

	if len(d) == 0 {
		return nil, errors.New("device not found")
	}

	t.Device = d[0]

	cfg, err := t.Device.Config(1)
	if err != nil {
		log.Error().Msgf("%s.Config(1): %v", t.Device, err)
		return nil, err
	}

	intf, err := cfg.Interface(0, 0)
	if err != nil {
		cfg.Close()
		log.Error().Msgf("%s.DefaultInterface(): %v", t.Device, err)
		return nil, err
	}

	ep_out, err := intf.OutEndpoint(1)
	if err != nil {
		cfg.Close()
		intf.Close()
		log.Fatal().Msgf("%s.OutEndpoint(1): %v", intf, err)
		return nil, err
	}

	ep_in, err := intf.InEndpoint(1)
	if err != nil {
		cfg.Close()
		intf.Close()
		log.Fatal().Msgf("%s.InEndpoint(1): %v", intf, err)
		return nil, err
	}

	t.Config = cfg
	t.Interface = intf
	t.EndpointOut = ep_out
	t.EndpointIn = ep_in

	return t, nil
}

func (c *USB_DEV) Close() {
	if c.Interface != nil {
		c.Interface.Close()
	}
	if c.Config != nil {
		c.Config.Close()
	}
	if c.Device != nil {
		c.Device.Close()
	}
}

func enumerate() {
	usb_devices_mutex.Lock()
	defer usb_devices_mutex.Unlock()
	all_found := map[string]bool{}

	// List all devices.
	ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		if desc.Path == nil || len(desc.Path) == 0 {
			return false // skip
		}

		found := false
		sid := GetUSB_ID(desc)
		for _, conn := range usb_devices {
			if conn.USB_ID == sid {
				all_found[sid] = true
				found = true
				break
			}
		}

		if !found {
			usb_devices = append(usb_devices, &USB_DEV{
				USB_ID: sid,
				Device: nil,
			})
			all_found[sid] = true

			v, p := ResolveVendorProduct(uint16(desc.Vendor), uint16(desc.Product))

			log.Debug().Msgf("New device connected: %s %s %s", sid, v, p)
			bus.Send("usb", "connected", &bus.B_UsbConnected{
				USB_ID:  sid,
				Vendor:  v,
				Product: p,
			})
		}

		return false
	})

	// Close all devices that are not found
	for i := 0; i < len(usb_devices); i++ {
		if !all_found[usb_devices[i].USB_ID] {
			usb_devices[i].Close()
			bus.Send("usb", "disconnected", &bus.B_UsbDisconnected{
				USB_ID: usb_devices[i].Device.String(),
			})
			usb_devices = append(usb_devices[:i], usb_devices[i+1:]...)
			i--
		}
	}
}
