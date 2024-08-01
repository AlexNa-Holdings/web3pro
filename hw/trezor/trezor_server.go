package trezor

import (
	"sync"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/signer/trezorproto"
	"github.com/rs/zerolog/log"
)

// connected Trezor
type Trezor struct {
	USB_ID string
	Name   string
	*trezorproto.Features
}

var trezors = []*Trezor{}
var trezors_mutex = &sync.Mutex{}

func Loop() {
	ch := bus.Subscribe("hw", "usb")
	for msg := range ch {
		if msg.RespondTo != 0 {
			continue // ignore responses
		}

		switch msg.Topic {
		case "usb":
			switch msg.Type {
			case "connected":
				m, ok := msg.Data.(*bus.B_UsbConnected)
				if !ok {
					log.Error().Msg("Loop: Invalid usb connected data")
					continue
				}
				connected(m)

			case "disconnected":
				m, ok := msg.Data.(*bus.B_UsbDisconnected)
				if !ok {
					log.Error().Msg("Loop: Invalid usb disconnected data")
					continue
				}
				disconnected(m)
			}
		case "hw":
		}
	}
}

func remove(usb_id string) {
	trezors_mutex.Lock()
	defer trezors_mutex.Unlock()

	for i, t := range trezors {
		if t.USB_ID == usb_id {
			trezors = append(trezors[:i], trezors[i+1:]...)
			return
		}
	}
}

func add(t *Trezor) {
	remove(t.USB_ID) // if reconnected

	trezors_mutex.Lock()
	defer trezors_mutex.Unlock()

	trezors = append(trezors, t)
}

func connected(m *bus.B_UsbConnected) {
	if m.Vendor != "SatoshiLabs" {
		return
	}

	t, err := init_trezor(m.USB_ID)
	if err != nil {
		log.Error().Err(err).Msg("Error initializing trezor")
		return
	}

	add(t)
}

func disconnected(m *bus.B_UsbDisconnected) {
	remove(m.USB_ID)
}
