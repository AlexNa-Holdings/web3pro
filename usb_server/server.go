package usb_server

import (
	"sync"
	"time"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/google/gousb"
	"github.com/rs/zerolog/log"
)

var ctx *gousb.Context

type Connection struct {
	Type   string
	Name   string
	Device *gousb.Device
}

var connections = []*Connection{}
var connections_mutex = &sync.Mutex{}

// Vendor and Product IDs
const (
	VID_Trezor1 = 0x534c
	VID_Trezor2 = 0x1209
	VID_Ledger  = 0x2c97
)

const (
	TypeT1Hid    = 0
	TypeT1Webusb = 1

	TypeT1WebusbBoot    = 2
	TypeT2              = 3
	TypeT2Boot          = 4
	TypeEmulator        = 5
	ProductT2Bootloader = 0x53C0
	ProductT2Firmware   = 0x53C1
	LedgerNanoS         = 0x0001
	LedgerNanoX         = 0x0004
	LedgerBlue          = 0x0000
	ProductT1Firmware   = 0x0001
)

func Init() {
	go MessageLoop()
}

func MessageLoop() {
	ctx = gousb.NewContext()
	defer ctx.Close()

	ch := bus.Subscribe("usb")
	enum_ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case msg := <-ch:
			if msg.RespondTo != 0 {
				continue // ignore responses
			}

			switch msg.Type {
			case "list":
				list := bus.B_UsbList_Response{}

				for _, conn := range connections {
					list = append(list, bus.B_UsbList_Device{
						Path: conn.Device.String(),
						Name: conn.Name,
						Type: conn.Type,
					})
				}

				msg.Respond(list, nil)
			}
		case <-enum_ticker.C:
			enumerate()
		}
	}
}

func enumerate() {
	connections_mutex.Lock()
	defer connections_mutex.Unlock()

	// Open all devices.
	devices, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return IsSupportedDevice(uint16(desc.Vendor), uint16(desc.Product))
	})

	if err != nil {
		log.Error().Err(err).Msg("enumerate: Error opening devices")

		for _, conn := range connections {
			conn.Device.Close()
		}
		connections = []*Connection{}
		return
	}

	all_found := map[string]bool{}

	for _, dev := range devices {

		sid := dev.String()
		found := false
		for _, conn := range connections {
			if conn.Device.String() == sid {
				all_found[sid] = true
				found = true
				break
			}
		}

		if !found {
			// New device
			connections = append(connections, &Connection{
				Type:   USBDeviceType(uint16(dev.Desc.Vendor), uint16(dev.Desc.Product)),
				Name:   dev.Desc.String(),
				Device: dev,
			})
			all_found[sid] = true
		}
	}

	// Close all devices that are not found
	for i := 0; i < len(connections); i++ {
		if !all_found[connections[i].Device.String()] {
			connections[i].Device.Close()
			connections = append(connections[:i], connections[i+1:]...)
			i--
		}
	}
}

func IsTrezor1(vid uint16, pid uint16) bool {
	return vid == VID_Trezor1 && pid == ProductT1Firmware
}

func IsTrezor2(vid uint16, pid uint16) bool {
	return vid == VID_Trezor2 && (pid == ProductT2Firmware || pid == ProductT2Bootloader)
}

func IsTrezor(vid uint16, pid uint16) bool {
	return IsTrezor1(vid, pid) || IsTrezor2(vid, pid)
}

func IsLedger(vid uint16, pid uint16) bool {
	return vid == VID_Ledger
}

func IsSupportedDevice(vid uint16, pid uint16) bool {
	return IsTrezor(vid, pid) || IsLedger(vid, pid)
}

func USBDeviceType(vid uint16, pid uint16) string {
	if IsTrezor(vid, pid) {
		return "trezor"
	}
	if IsLedger(vid, pid) {
		return "ledger"
	}
	return ""
}
