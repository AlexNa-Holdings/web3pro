package trezor

import (
	"sync"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/hw/trezor/trezorproto"
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

const TRZ = "trezor"

func Loop() {
	ch := bus.Subscribe("signer", "usb")
	for msg := range ch {
		if msg.RespondTo != 0 {
			continue // ignore responses
		}
		go process(msg)
	}
}

func process(msg *bus.Message) {
	switch msg.Topic {
	case "usb":
		switch msg.Type {
		case "connected":
			if m, ok := msg.Data.(*bus.B_UsbConnected); ok && m.Vendor == "SatoshiLabs" {
				connected(m)
			}
		case "disconnected":
			if m, ok := msg.Data.(*bus.B_UsbDisconnected); ok {
				disconnected(m)
			} else {
				log.Error().Msg("Loop: Invalid usb disconnected data")
			}
		}
	case "signer":
		switch msg.Type {
		case "is-connected":
			if m, ok := msg.Data.(*bus.B_SignerIsConnected); ok {
				if m.Type != TRZ {
					msg.Respond(&bus.B_SignerIsConnected_Response{Connected: find_by_name(m.Name) != nil}, nil)
				}
			} else {
				log.Error().Msg("Loop: Invalid hw is-connected data")
			}
		case "get-addresses":
			if m, ok := msg.Data.(*bus.B_SignerGetAddresses); ok {
				if m.Type == TRZ {
					msg.Respond(get_addresses(msg))
				}
			} else {
				log.Error().Msg("Loop: Invalid hw get-addresses data")
			}
		case "list":
			if m, ok := msg.Data.(*bus.B_SignerList); ok {
				if m.Type == TRZ {
					msg.Respond(&bus.B_SignerList_Response{Names: list()}, nil)
				}
			} else {
				log.Error().Msg("Loop: Invalid trezor list data")
			}
		}
	}
}

func list() []string {
	trezors_mutex.Lock()
	defer trezors_mutex.Unlock()

	var names []string
	for _, t := range trezors {
		names = append(names, t.Name)
	}

	return names
}

func remove(usb_id string) {
	trezors_mutex.Lock()
	defer trezors_mutex.Unlock()

	for i, t := range trezors {
		if t.USB_ID == usb_id {
			cmn.Notifyf("Trezor disconnected: %s", t.Name)
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
	log.Debug().Msgf("Trezor Connected: %s %s", m.Vendor, m.Product)

	t, err := init_trezor(m.USB_ID)
	if err != nil {
		log.Error().Err(err).Msg("Error initializing trezor")
		return
	}

	add(t)
	cmn.Notifyf("Trezor connected: %s", t.Name)
}

func disconnected(m *bus.B_UsbDisconnected) {
	remove(m.USB_ID)
}

func find_by_name(name []string) *Trezor {
	trezors_mutex.Lock()
	defer trezors_mutex.Unlock()

	for _, t := range trezors {
		for _, n := range name {
			if t.Name == n {
				return t
			}
		}
	}

	return nil
}

func provide_device(n []string) *Trezor {

	if len(n) == 0 {
		log.Error().Msg("Open: No device name provided")
		return nil
	}

	t := find_by_name(n)
	if t != nil {
		return t
	}

	name := n[0]
	copies := ""
	if len(n) > 1 {
		copies = "\n or one of the copies:\n<u><b>"
		for i, c := range n {
			copies += c
			if i < len(n)-1 {
				copies += ", "
			}
		}
		copies += "</b></u>"
	}

	bus.Fetch("ui", "hail", &bus.B_Hail{
		Title: "Connect Trezor",
		Template: `<c><w>
Please connect your Trezor device:

<u><b>` + name + `</b></u>` + copies + `

<button text:Cancel>`,
		OnTick: func(m *bus.Message, tick int) {
			t = find_by_name(n)
			if t != nil {
				bus.Send("ui", "remove-hail", m)
			}
		},
	})

	return t
}
