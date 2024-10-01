package ledger

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

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

var CLEANING_APDU = APDU{0xe0, 0x50, 0x00, 0x00}
var GET_DEVICE_NAME_APDU = APDU{0xe0, 0xd2, 0x00, 0x00}
var GET_INFO_APDU = APDU{0xb0, 0x01, 0x00, 0x00}
var QUIT_APP_APDU = APDU{0xb0, 0xa7, 0x00, 0x00}
var LAUNCH_APP_APDU = APDU{0xe0, 0xd8, 0x00, 0x00}
var GET_ADDRESS_APDU = APDU{0xe0, 0x02, 0x00, 0x00}
var SIGN_TX_APDU = APDU{0xe0, 0x04, 0x00, 0x00}
var SIGN_MSG_APDU = APDU{0xe0, 0x0c, 0x00, 0x01}
var SIGN_MSG_PERSONAL_APDU = APDU{0xe0, 0x08, 0x00, 0x00}

var STRUCT_DEF_NAME = APDU{0xe0, 0x1a, 0x00, 0x00}
var STRUCT_DEF_FIELD = APDU{0xe0, 0x1a, 0x00, 0xff}

var STRUCT_IMPL_ROOT = APDU{0xe0, 0x1c, 0x00, 0x00}
var STRUCT_IMPL_ARRAY = APDU{0xe0, 0x1c, 0x00, 0x0f}
var STRUCT_IMPL_FIELD = APDU{0xe0, 0x1c, 0x00, 0xff}

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
				log.Debug().Msg("Ledger usb connected")
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
		w := cmn.CurrentWallet
		if w == nil {
			msg.Respond(nil, errors.New("no wallet"))
			return
		}

		switch msg.Type {
		case "is-connected":
			if m, ok := msg.Data.(*bus.B_SignerIsConnected); ok {
				if m.Type == LDG {
					msg.Respond(&bus.B_SignerIsConnected_Response{
						Connected: find_by_name(w.GetSignerWithCopies(m.Name)) != nil},
						nil,
					)
				}
			} else {
				log.Error().Msg("Loop: Invalid hw is-connected data")
			}
		case "get-addresses":
			if m, ok := msg.Data.(*bus.B_SignerGetAddresses); ok {
				if m.Type == LDG {
					msg.Respond(get_addresses(m))
				}
			} else {
				log.Error().Msg("Loop: Invalid hw get-addresses data")
			}
		case "list":
			if m, ok := msg.Data.(*bus.B_SignerList); ok {
				if m.Type == LDG {
					msg.Respond(&bus.B_SignerList_Response{Names: list()}, nil)
				}
			} else {
				log.Error().Msg("Loop: Invalid ledger list data")
			}
		case "sign-typed-data-v4":
			m, ok := msg.Data.(*bus.B_SignerSignTypedData_v4)
			if !ok {
				log.Error().Msg("Loop: Invalid hw sign-typed-data-v4 data")
				msg.Respond(nil, errors.New("invalid data"))
				return
			}

			if m.Type == LDG {
				msg.Respond(signTypedData_v4(msg))
			}
		case "sign-tx":
			m, ok := msg.Data.(*bus.B_SignerSignTx)
			if !ok {
				log.Error().Msg("Loop: Invalid hw sign-tx data")
				msg.Respond(nil, errors.New("invalid data"))
				return
			}

			if m.Type == LDG {
				msg.Respond(signTx(msg))
			}
		case "sign":
			m, ok := msg.Data.(*bus.B_SignerSign)
			if !ok {
				log.Error().Msg("Loop: Invalid hw sign data")
				msg.Respond(nil, errors.New("invalid data"))
				return
			}

			if m.Type == LDG {
				msg.Respond(sign(msg))
			}
		}
	}
}

func list() []string {
	ledgers_mutex.Lock()
	defer ledgers_mutex.Unlock()

	var names []string
	for _, t := range ledgers {
		if t.Name == "" {
			bus.Send("usb", "connected", &bus.B_UsbConnected{USB_ID: t.USB_ID}) // trigger name initialization
		} else {
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
			bus.Send("ui", "notify", fmt.Sprintf("Ledger disconnected: %s", t.Name))
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
	bus.Send("ui", "notify", fmt.Sprintf("Ledger connected: %s", t.Name))
}

func disconnected(m *bus.B_UsbDisconnected) {
	remove(m.USB_ID)
}

func find_by_name(name []*cmn.Signer) *Ledger {
	log.Debug().Msgf("find_by_name: %v", name)

	ledgers_mutex.Lock()
	defer ledgers_mutex.Unlock()

	for _, t := range ledgers {
		for _, n := range name {
			if t.Name == n.Name {
				return t
			}
		}
	}

	log.Debug().Msg("find_by_name: not found")

	// if not found let's try to initialize those without a name
	//check if there are not initialized ledgers
	for _, t := range ledgers {
		if t.Name == "" {

			log.Debug().Msgf("find_by_name: initializing %s", t.USB_ID)

			n, err := getName(t.USB_ID)
			if err != nil {
				log.Error().Err(err).Msg("Error initializing ledger")
			}
			t.Name = n
			bus.Send("signer", "connected", &bus.B_SignerConnected{Type: LDG, Name: t.Name})
			bus.Send("ui", "notify", fmt.Sprintf("Ledger connected: %s", t.Name))
			for _, n := range name {
				if t.Name == n.Name {
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

func provide_device(sn string) *Ledger {
	w := cmn.CurrentWallet
	if w == nil {
		return nil
	}

	s_list := w.GetSignerWithCopies(sn)

	if len(s_list) == 0 {
		log.Error().Msg("Open: No device name provided")
		return nil
	}

	t := find_by_name(s_list)
	if t != nil {
		return t
	}

	name := s_list[0].Name
	copies := ""
	if len(s_list) > 1 {
		copies = "\n or one of the copies:\n<u><b>"
		for i, c := range s_list {
			copies += c.Name
			if i < len(s_list)-1 {
				copies += ", "
			}
		}
		copies += "</b></u>"
	}

	bus.Fetch("ui", "hail", &bus.B_Hail{
		Title:     "Connect Ledger",
		Priorized: true,
		Template: `<c><w>
Please connect your Ledger device:

<u><b>` + name + `</b></u>` + copies + `

<button text:Cancel>`,
		OnTick: func(m *bus.Message, tick int) {
			t = find_by_name(s_list)
			if t != nil {
				bus.Send("ui", "remove-hail", m)
			}
		},
	})

	return t
}

func provide_eth_app(usb_id string, needed_app string) error {

	log.Debug().Msgf("provide_eth_app: %s %s", usb_id, needed_app)

	r, err := call(usb_id, &GET_INFO_APDU, nil, generalHail, 5)
	if err != nil {
		log.Error().Err(err).Msgf("provide_eth_app: Error getting device name: %s", usb_id)
		return err
	}

	name, ver, err := parseGetInfoResponse(r)
	if err != nil {
		log.Error().Err(err).Msg("provide_eth_app: Error parsing get info response")
		return err
	}

	log.Trace().Msgf("provide_eth_app: Name: %s, Version: %s", name, ver)

	if name != needed_app {
		if needed_app == "BOLOS" {
			_, err := call(usb_id, &QUIT_APP_APDU, nil, generalHail, 5)
			if err != nil {
				log.Error().Err(err).Msgf("provide_eth_app: Error quitting app: %s", usb_id)
				return err
			}
		} else {
			_, err := call(usb_id, &LAUNCH_APP_APDU, []byte(needed_app), generalHail, 0)
			if err != nil {
				log.Error().Err(err).Msgf("provide_eth_app: Error quitting app: %s", usb_id)
				return err
			}
		}

		// wait for the device reconnected
		err = waitForReconnect(usb_id)
		if err != nil {
			log.Error().Err(err).Msgf("provide_eth_app: Error waiting for reconnect: %s",
				usb_id)
			return err
		}
	}
	return nil
}

func waitForReconnect(usb_id string) error {
	const timeout = 500 * time.Millisecond
	const tries = 10
	for i := 0; i < tries; i++ {

		<-time.After(timeout)

		m := bus.Fetch("usb", "is_connected", &bus.B_UsbIsConnected{USB_ID: usb_id})
		if m.Error == nil {
			res, ok := m.Data.(*bus.B_UsbIsConnected_Response)
			if ok {
				if res.Connected {
					log.Trace().Msg("waitForReconnect: Reconnected")
					return nil
				}
			} else {
				log.Error().Msg("waitForReconnect: Invalid message data. Expected B_SignerIsConnected_Response")
			}
		}
	}
	return fmt.Errorf("waitForReconnect: Timeout")
}

func parseGetInfoResponse(data []byte) (string, string, error) {
	if len(data) < 3 {
		return "", "", fmt.Errorf("response too short")
	}

	// The second byte is the length of the name
	nameLength := int(data[1])
	if len(data) < 2+nameLength+1 {
		return "", "", fmt.Errorf("response too short for name length")
	}

	// Extract the name
	nameBytes := data[2 : 2+nameLength]
	name := string(nameBytes)

	// The byte after the name length is the length of the version
	versionStart := 2 + nameLength
	if len(data) < versionStart+1 {
		return "", "", fmt.Errorf("response too short for version length")
	}

	versionLength := int(data[versionStart])
	if len(data) < versionStart+1+versionLength {
		return "", "", fmt.Errorf("response too short for version length")
	}

	// Extract the version
	versionBytes := data[versionStart+1 : versionStart+1+versionLength]
	version := string(versionBytes)

	return name, version, nil
}
