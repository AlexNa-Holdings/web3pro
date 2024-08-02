package trezor

import (
	"sync"

	"github.com/AlexNa-Holdings/web3pro/bus"
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

func Loop() {
	ch := bus.Subscribe("hw", "usb")
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
			m, ok := msg.Data.(*bus.B_UsbConnected)
			if !ok {
				log.Error().Msg("Loop: Invalid usb connected data")
				return
			}
			connected(m)

		case "disconnected":
			m, ok := msg.Data.(*bus.B_UsbDisconnected)
			if !ok {
				log.Error().Msg("Loop: Invalid usb disconnected data")
				return
			}
			disconnected(m)
		}
	case "hw":
		switch msg.Type {
		case "is-connected":
			m, ok := msg.Data.(*bus.B_HwIsConnected)
			if !ok {
				log.Error().Msg("Loop: Invalid hw is-connected data")
				return
			}

			if m.Type == "trezor" {
				msg.Respond(&bus.B_HwIsConnected_Response{Connected: find_by_name(m.Name) != nil}, nil)
			}

		case "get-addresses":
			m, ok := msg.Data.(*bus.B_HwGetAddresses)
			if !ok {
				log.Error().Msg("Loop: Invalid hw get-addresses data")
				return
			}

			if m.Type == "trezor" {
				msg.Respond(get_addresses(m))
			}
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

	log.Debug().Msgf("Connected: %s %s", m.Vendor, m.Product)

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
		OnTick: func(h *bus.B_Hail, tick int) {
			t = find_by_name(n)
			if t != nil {
				bus.Send("ui", "remove_hail", h)
			}
		},
	})

	return t
}
