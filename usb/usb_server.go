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
	USB_ID      string
	Vendor      string
	Device      *gousb.Device
	Config      *gousb.Config
	Interface   *gousb.Interface
	EndpointOut *gousb.OutEndpoint
	EndpointIn  *gousb.InEndpoint

	connected      bool
	disconnectTime time.Time
}

const USB_IDS = "usb.ids"

var operation_mutex = &sync.Mutex{}

var usb_devices = []*USB_DEV{}
var usb_devices_mutex = &sync.Mutex{}

var SHORT_DISCONNECT_PERIOD = 5 * time.Second

var RATE_SLOW = 3 * time.Second
var RATE_FAST = 1 * time.Second
var enum_ticker_rate = RATE_SLOW
var enum_ticker = time.NewTicker(enum_ticker_rate)

func Init() {
	init_usb_ids()
	go Loop()
}

func setRate(rate time.Duration) {
	if rate != enum_ticker_rate {
		enum_ticker.Reset(rate)
	}
}

func Loop() {
	ctx = gousb.NewContext()
	defer ctx.Close()

	ch := bus.Subscribe("usb")

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
	operation_mutex.Lock()
	defer operation_mutex.Unlock()

	list := bus.B_UsbList_Response{}

	ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {

		if len(desc.Path) == 0 {
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
		log.Error().Msg("usb.write: Invalid message data. Expected B_UsbWrite")
		return bus.ErrInvalidMessageData
	}

	conn, err := OpenDevice(req.USB_ID)
	if err != nil {
		log.Error().Msg("usb.write: Device not found")
		return fmt.Errorf("usb.write: device not found: %s", req.USB_ID)
	}
	if conn.EndpointOut == nil {
		log.Error().Msg("usb.write: EndpointOut is nil")
		return fmt.Errorf("usb.write: EndpointOut not initialized for device: %s", req.USB_ID)
	}
	if msg.TimerID == 0 {
		log.Error().Msg("usb.write: TimerID is required for USD writing")
		return fmt.Errorf("usb.write: TimerID is required")
	}
	cancel_ctx, cancel_func := context.WithCancel(context.Background())

	msg.OnCancel = func(m *bus.Message) {
		cancel_func()
	}

	operation_mutex.Lock()
	log.Debug().Msg("Before write")
	_, err = conn.EndpointOut.WriteContext(cancel_ctx, req.Data)
	log.Debug().Msg("After write")
	operation_mutex.Unlock()

	if err != nil {
		log.Error().Err(err).Msg("usb.write: Error writing to device")

		if strings.Contains(err.Error(), "no device") {
			log.Error().Msg("usb.write: Device not found")
			removeUSBDevice(req.USB_ID)
		}

		return err
	}

	msg.OnCancel = nil
	return nil
}

func read(msg *bus.Message) (*bus.B_UsbRead_Response, error) {
	req, ok := msg.Data.(*bus.B_UsbRead)
	if !ok {
		log.Error().Msg("Invalid message data. Expected B_UsbRead")
		return nil, bus.ErrInvalidMessageData
	}
	conn, err := OpenDevice(req.USB_ID)
	if err != nil {
		log.Error().Msg("Device not found")
		return nil, fmt.Errorf("device not found: %s", req.USB_ID)
	}
	if conn.EndpointIn == nil {
		log.Error().Msg("usb.read: EndpointIn is nil")
		return nil, fmt.Errorf("usb.read: EndpointIn not initialized for device: %s", req.USB_ID)
	}

	if msg.TimerID == 0 {
		log.Error().Msg("TimerID is required for USD reading")
		return nil, errors.New("TimerID is required")
	}

	cancel_ctx, cancel_func := context.WithCancel(context.Background())
	msg.OnCancel = func(m *bus.Message) {
		cancel_func()
	}

	data := make([]byte, conn.EndpointIn.Desc.MaxPacketSize)

	log.Debug().Msgf("Reading from device Timer: %d", msg.TimerID)
	operation_mutex.Lock()
	log.Debug().Msg("Before read")
	n, err := conn.EndpointIn.ReadContext(cancel_ctx, data)
	log.Debug().Msg("After read")
	operation_mutex.Unlock()
	log.Debug().Msg("After read unlock")

	if err != nil {
		log.Error().Err(err).Msg("Error reading from device")

		if strings.Contains(err.Error(), "no device") {
			log.Error().Msg("usb.write: Device not found")
			removeUSBDevice(req.USB_ID)
		}

		return nil, err
	}

	msg.OnCancel = nil
	return &bus.B_UsbRead_Response{Data: data[:n]}, nil
}

func is_connected(usb_id string) bool {
	enumerate()
	conn := getUSBDevice(usb_id)
	if conn != nil {
		return conn.connected
	}
	return false
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
		case "is_connected":
			if m, ok := msg.Data.(*bus.B_UsbIsConnected); ok {
				msg.Respond(&bus.B_UsbIsConnected_Response{Connected: is_connected(m.USB_ID)}, nil)
			} else {
				log.Error().Msg("Invalid message data. Expected B_UsbIsConnected")
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

	// If device is already properly opened with endpoints, return it
	if t.Device != nil && t.EndpointIn != nil && t.EndpointOut != nil {
		return t, nil // already opened
	}

	// If device was partially opened (failed previously), close it first
	if t.Device != nil {
		t.Close()
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

	device := d[0]

	// On macOS, SetAutoDetach can cause issues - skip it
	// The HID driver detachment is handled differently on macOS
	// if err := device.SetAutoDetach(true); err != nil {
	// 	log.Error().Err(err).Msg("SetAutoDetach(true)")
	// }

	cfg, err := device.Config(1)
	if err != nil {
		device.Close()
		log.Error().Msgf("%s.Config(1): %v", device, err)
		return nil, err
	}

	// Try to claim interface - on macOS, may need to try different interfaces
	var iface *gousb.Interface
	var ifaceErr error
	for ifaceNum := 0; ifaceNum < 3; ifaceNum++ {
		iface, ifaceErr = cfg.Interface(ifaceNum, 0)
		if ifaceErr == nil {
			log.Trace().Msgf("Successfully claimed interface %d", ifaceNum)
			break
		}
		log.Trace().Msgf("Failed to claim interface %d: %v", ifaceNum, ifaceErr)
	}
	if iface == nil {
		cfg.Close()
		device.Close()
		log.Error().Err(ifaceErr).Msgf("Failed to claim any interface")
		return nil, ifaceErr
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
		iface.Close()
		cfg.Close()
		device.Close()
		log.Error().Msg("No endpoints found")
		return nil, errors.New("no endpoints found")
	}

	// Only set everything on success
	t.Device = device
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
	operation_mutex.Lock()
	defer operation_mutex.Unlock()

	presented := map[string]bool{}

	// List all devices.
	ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		if desc.Path == nil || len(desc.Path) == 0 {
			return false // skip
		}

		sid := GetUSB_ID(desc)

		found_dev := getUSBDevice(sid)
		presented[sid] = true

		if found_dev != nil {

			if !found_dev.connected {
				found_dev.connected = true
				log.Trace().Msgf("Device %s Reconnected.", sid)
			}
		} else {
			v, p := ResolveVendorProduct(uint16(desc.Vendor), uint16(desc.Product))
			log.Trace().Msgf("Device %s connected: %s %s", sid, v, p)

			addUSBDevice(sid, &USB_DEV{
				Vendor:    v,
				USB_ID:    sid,
				Device:    nil,
				connected: true,
			})

			bus.Send("usb", "connected", &bus.B_UsbConnected{
				USB_ID:  sid,
				Vendor:  v,
				Product: p,
			})
		}

		return false
	})

	not_presented := []*USB_DEV{}
	usb_devices_mutex.Lock()
	for _, dev := range usb_devices {
		if !presented[dev.USB_ID] {
			not_presented = append(not_presented, dev)
		}
	}
	usb_devices_mutex.Unlock()

	// Close all devices that are not found
	for _, dev := range not_presented {
		if dev.connected {
			dev.Close()
			dev.connected = false
			dev.disconnectTime = time.Now()
			log.Trace().Msgf("Device %s marked as disconnected", dev.USB_ID)
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

	n_just_disconnected := 0
	delete_immediately := []*USB_DEV{}

	usb_devices_mutex.Lock()
	for _, dev := range usb_devices {
		if !dev.connected {

			if dev.Vendor != "Ledger" {
				delete_immediately = append(delete_immediately, dev)
			} else { // delay disconnect
				n_just_disconnected++
			}
		}
	}
	usb_devices_mutex.Unlock()

	for _, dev := range delete_immediately {
		log.Trace().Msgf("Device %s disconnected", dev.USB_ID)
		bus.Send("usb", "disconnected", &bus.B_UsbDisconnected{
			USB_ID: dev.USB_ID,
		})
		removeUSBDevice(dev.USB_ID)
	}

	if n_just_disconnected == 0 {
		setRate(RATE_SLOW)
	} else {
		setRate(RATE_FAST)
	}
}
