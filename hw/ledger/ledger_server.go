package ledger

import (
	"encoding/binary"
	"sync"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/rs/zerolog/log"
)

// connected Ledger
type Ledger struct {
	USB_ID string
	Name   string
}

type APDU struct {
	cla     byte
	op_code byte
	p1      byte
	p2      byte
}

var ledgers = []*Ledger{}
var ledgers_mutex = &sync.Mutex{}

const LDG = "ledger"

var codec = binary.BigEndian

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
			if m, ok := msg.Data.(*bus.B_UsbConnected); ok && m.Vendor == "Ledger" {
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
			if m, ok := msg.Data.(*bus.B_SignerIsConnected); ok && m.Type == LDG {
				msg.Respond(&bus.B_SignerIsConnected_Response{Connected: find_by_name(m.Name) != nil}, nil)
			} else {
				log.Error().Msg("Loop: Invalid hw is-connected data")
			}
		case "get-addresses":
			if m, ok := msg.Data.(*bus.B_SignerGetAddresses); ok && m.Type == LDG {
				msg.Respond(get_addresses(m))
			} else {
				log.Error().Msg("Loop: Invalid hw get-addresses data")
			}
		case "list":

			log.Debug().Msgf("List received %v", msg.Data)

			if m, ok := msg.Data.(*bus.B_SignerList); ok && m.Type == LDG {
				msg.Respond(&bus.B_SignerList_Response{Names: list()}, nil)
			} else {
				log.Error().Msg("Loop: Invalid hw list data")
			}
		}
	}
}

func list() []string {

	log.Debug().Msg("List")
	ledgers_mutex.Lock()
	defer ledgers_mutex.Unlock()

	var names []string
	for _, t := range ledgers {
		if t.Name == "" {
			n, err := getName(t.USB_ID)
			if err != nil {
				log.Error().Err(err).Msg("Error initializing ledger")
			} else {
				t.Name = n
			}
		}

		if t.Name != "" {
			names = append(names, t.Name)
		}

	}

	return names
}

func remove(usb_id string) {
	ledgers_mutex.Lock()
	defer ledgers_mutex.Unlock()

	for i, t := range ledgers {
		if t.USB_ID == usb_id {
			cmn.Notifyf("Ledger disconnected: %s", t.Name)
			ledgers = append(ledgers[:i], ledgers[i+1:]...)
			return
		}
	}
}

func add(t *Ledger) {
	remove(t.USB_ID) // if reconnected

	ledgers_mutex.Lock()
	defer ledgers_mutex.Unlock()

	ledgers = append(ledgers, t)
}

func connected(m *bus.B_UsbConnected) {
	log.Debug().Msgf("Ledger Connected: %s %s", m.Vendor, m.Product)

	t := &Ledger{
		USB_ID: m.USB_ID,
	}

	add(t)

	n, err := getName(m.USB_ID)
	if err != nil {
		log.Error().Err(err).Msg("Error initializing ledger")
		return
	}

	t.Name = n
	bus.Send("signer", "connected", &bus.B_SignerConnected{Type: LDG, Name: t.Name})
	cmn.Notifyf("Ledger connected: %s", t.Name)
}

func disconnected(m *bus.B_UsbDisconnected) {
	remove(m.USB_ID)
}

func find_by_name(name []string) *Ledger {
	ledgers_mutex.Lock()
	defer ledgers_mutex.Unlock()

	for _, t := range ledgers {
		for _, n := range name {
			if t.Name == n {
				return t
			}
		}
	}

	// if not found let's try to initialize those without a name
	//check if there are not initialized ledgers
	for _, t := range ledgers {
		if t.Name == "" {
			n, err := getName(t.USB_ID)
			if err != nil {
				log.Error().Err(err).Msg("Error initializing ledger")
			}
			t.Name = n
			bus.Send("signer", "connected", &bus.B_SignerConnected{Type: LDG, Name: t.Name})
			cmn.Notifyf("Ledger connected: %s", t.Name)
			for _, n := range name {
				if t.Name == n {
					return t
				}
			}
		}
	}

	return nil
}

func find_by_usb_id(usb_id string) *Ledger {
	ledgers_mutex.Lock()
	defer ledgers_mutex.Unlock()

	for _, t := range ledgers {
		if t.USB_ID == usb_id {
			return t
		}
	}
	return nil
}

func provide_device(n []string) *Ledger {

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
		Title: "Connect Ledger",
		Template: `<c><w>
Please connect your Ledger device:

<u><b>` + name + `</b></u>` + copies + `

<button text:Cancel>`,
		OnTick: func(h *bus.B_Hail, tick int) {
			t = find_by_name(n)
			if t != nil {
				bus.Send("ui", "remove-hail", h)
			}
		},
	})

	return t
}
