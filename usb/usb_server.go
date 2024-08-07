package usb

import (
	"context"
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

	connected      bool
	disconnectTime time.Time
}

const USB_IDS = "usb.ids"

var usb_devices = []*USB_DEV{}
var usb_devices_mutex = &sync.Mutex{}

var rw_cancels = map[int]context.CancelFunc{} // timer id -> cancel
var rw_cancels_mutex = &sync.Mutex{}

var SHORT_DISCONNECT_PERIOD = 5 * time.Second

func Init() {
	init_usb_ids()
	go Loop()
}

func addCancel(id int, cancel context.CancelFunc) {
	rw_cancels_mutex.Lock()
	defer rw_cancels_mutex.Unlock()
	rw_cancels[id] = cancel
}

func doCancel(id int) {
	rw_cancels_mutex.Lock()
	defer rw_cancels_mutex.Unlock()

	c, ok := rw_cancels[id]
	if ok {
		c()
	}
	delete(rw_cancels, id)
}

func Loop() {
	ctx = gousb.NewContext()
	defer ctx.Close()

	ch := bus.Subscribe("usb", "timer")
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

func list() bus.B_UsbList_Response {
	list := bus.B_UsbList_Response{}

	ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {

		if desc.Path == nil || len(desc.Path) == 0 {
			return false // skip
		}

		conn := getUSBDevice(GetUSB_ID(desc))
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

	return list
}

func write(msg *bus.Message) error {
	req, ok := msg.Data.(*bus.B_UsbWrite)
	if !ok {
		log.Error().Msg("Invalid message data")
		return bus.ErrInvalidMessageData
	}

	conn, err := OpenDevice(req.USB_ID)
	if err != nil {
		log.Error().Msg("Device not found")
		return fmt.Errorf("device not found: %s", req.USB_ID)
	}
	if msg.TimerID == 0 {
		log.Error().Msg("TimerID is required for USD writing")
		return fmt.Errorf("TimerID is required")
	}
	cancel_ctx, cancel := context.WithCancel(context.Background())
	addCancel(msg.TimerID, cancel)
	_, err = conn.EndpointOut.WriteContext(cancel_ctx, req.Data)
	if err != nil {
		log.Error().Err(err).Msg("Error writing to device")
		return err
	}

	doCancel(msg.TimerID)
	return nil
}

func read(msg *bus.Message) (*bus.B_UsbRead_Response, error) {
	req, ok := msg.Data.(*bus.B_UsbRead)
	if !ok {
		log.Error().Msg("Invalid message data")
		return nil, bus.ErrInvalidMessageData
	}
	conn, err := OpenDevice(req.USB_ID)
	if err != nil {
		log.Error().Msg("Device not found")
		return nil, fmt.Errorf("device not found: %s", req.USB_ID)
	}

	if msg.TimerID == 0 {
		log.Error().Msg("TimerID is required for USD reading")
		return nil, errors.New("TimerID is required")
	}

	cancel_ctx, cancel := context.WithCancel(context.Background())
	addCancel(msg.TimerID, cancel)
	data := make([]byte, conn.EndpointIn.Desc.MaxPacketSize)
	n, err := conn.EndpointIn.ReadContext(cancel_ctx, data)
	if err != nil {
		log.Error().Err(err).Msg("Error reading from device")
		return nil, err
	}
	doCancel(msg.TimerID)

	return &bus.B_UsbRead_Response{Data: data[:n]}, nil
}

func process(msg *bus.Message) {
	switch msg.Topic {
	case "usb":
		switch msg.Type {
		case "list":
			msg.Respond(list(), nil)
		case "write":
			msg.Respond(nil, write(msg))
		case "read":
			msg.Respond(read(msg))
		}
	case "timer":
		switch msg.Type {
		case "done":
			if d, ok := msg.Data.(*bus.B_TimerDone); ok {
				doCancel(d.ID)
			} else {
				log.Error().Msg("Invalid message data")
			}
		}
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
	return fmt.Sprintf("vid=%s,pid=%s,bus=%d,path=%s", d.Vendor, d.Product, d.Bus, GetPath(d))
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

func getUSBDevice(id string) *USB_DEV {
	usb_devices_mutex.Lock()
	defer usb_devices_mutex.Unlock()

	for _, conn := range usb_devices {
		if conn.USB_ID == id {
			return conn
		}
	}
	return nil
}

func addUSBDevice(id string, dev *USB_DEV) {
	usb_devices_mutex.Lock()
	defer usb_devices_mutex.Unlock()

	usb_devices = append(usb_devices, dev)
}

func removeUSBDevice(id string) {
	usb_devices_mutex.Lock()
	defer usb_devices_mutex.Unlock()

	for i, conn := range usb_devices {
		if conn.USB_ID == id {
			usb_devices = append(usb_devices[:i], usb_devices[i+1:]...)
			return
		}
	}
}

func OpenDevice(id string) (*USB_DEV, error) {
	t := getUSBDevice(id)
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

	// Detach the kernel driver if necessary
	if err := t.Device.SetAutoDetach(true); err != nil {
		log.Error().Err(err).Msg("SetAutoDetach(true)")
	}

	iface, err := cfg.Interface(0, 0)
	if err != nil {
		cfg.Close()
		log.Error().Err(err).Msgf("Interface(0, 0)")
		return nil, err
	}

	var epIn *gousb.InEndpoint
	var epOut *gousb.OutEndpoint

	for _, ep := range iface.Setting.Endpoints {
		if ep.Direction == gousb.EndpointDirectionIn {
			epIn, _ = iface.InEndpoint(ep.Number)
		} else {
			epOut, _ = iface.OutEndpoint(ep.Number)
		}
	}

	if epIn == nil || epOut == nil {
		cfg.Close()
		log.Error().Msg("No endpoints found")
		return nil, errors.New("no endpoints found")
	}

	t.Config = cfg
	t.Interface = iface
	t.EndpointOut = epOut
	t.EndpointIn = epIn

	return t, nil
}

func (c *USB_DEV) Close() {
	if c.Interface != nil {
		c.Interface.Close()
		c.Interface = nil
	}
	if c.Config != nil {
		c.Config.Close()
		c.Config = nil
	}
	if c.Device != nil {
		c.Device.Close()
		c.Device = nil
	}
}

func enumerate() {
	all_found := map[string]bool{}

	// List all devices.
	ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		if desc.Path == nil || len(desc.Path) == 0 {
			return false // skip
		}

		sid := GetUSB_ID(desc)

		found_dev := getUSBDevice(sid)
		if found_dev != nil {
			all_found[sid] = true

			if !found_dev.connected {
				found_dev.connected = true
				log.Trace().Msgf("Device %s Address changed. Reconnected.", sid)
			}
		} else {
			addUSBDevice(sid, &USB_DEV{
				USB_ID:    sid,
				Device:    nil,
				connected: true,
			})

			v, p := ResolveVendorProduct(uint16(desc.Vendor), uint16(desc.Product))
			log.Trace().Msgf("Device %s connected: %s %s", sid, v, p)
			bus.Send("usb", "connected", &bus.B_UsbConnected{
				USB_ID:  sid,
				Vendor:  v,
				Product: p,
			})
		}

		return false
	})

	dev_not_found := []*USB_DEV{}
	usb_devices_mutex.Lock()
	for _, dev := range usb_devices {
		if !all_found[dev.USB_ID] {
			dev_not_found = append(dev_not_found, dev)
		}
	}
	usb_devices_mutex.Unlock()

	// Close all devices that are not found
	for _, dev := range dev_not_found {
		if dev.connected {
			dev.Close()
			dev.connected = false
			dev.disconnectTime = time.Now()
		} else {
			if time.Since(dev.disconnectTime) > SHORT_DISCONNECT_PERIOD {
				log.Trace().Msgf("Device %s disconnected", dev.USB_ID)
				bus.Send("usb", "disconnected", &bus.B_UsbDisconnected{
					USB_ID: dev.USB_ID,
				})
				removeUSBDevice(dev.USB_ID)
			}
		}
	}
}
