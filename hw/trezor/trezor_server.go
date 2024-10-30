package trezor

import (
	"errors"
	"fmt"
	"sync"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/hw/trezor/trezorproto"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/rs/zerolog/log"
)

// connected Trezor
type Trezor struct {
	*trezorproto.Features
	USB_ID string
	Name   string
	Pane   *TrezorPane
}

var trezors = []*Trezor{}
var trezors_mutex = &sync.Mutex{}

const TRZ = "trezor"

func Loop() {
	ch := bus.Subscribe("signer", "usb", "wallet")
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
	case "wallet":
		switch msg.Type {
		case "open":
			for _, t := range trezors {
				t.Pane.rebuidTemplate()
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
				if m.Type != TRZ {
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
		case "sign-typed-data-v4":
			m, ok := msg.Data.(*bus.B_SignerSignTypedData_v4)
			if !ok {
				log.Error().Msg("Loop: Invalid hw sign-typed-data-v4 data")
				msg.Respond(nil, errors.New("invalid data"))
				return
			}

			if m.Type == TRZ {
				msg.Respond(signTypedData_v4(msg))
			}
		case "sign-tx":
			m, ok := msg.Data.(*bus.B_SignerSignTx)
			if !ok {
				log.Error().Msg("Loop: Invalid hw sign-tx data")
				msg.Respond(nil, errors.New("invalid data"))
				return
			}

			if m.Type == TRZ {
				msg.Respond(signTx(msg))
			}
		case "sign":
			m, ok := msg.Data.(*bus.B_SignerSign)
			if !ok {
				log.Error().Msg("Loop: Invalid hw sign data")
				msg.Respond(nil, errors.New("invalid data"))
				return
			}

			if m.Type == TRZ {
				msg.Respond(sign(msg))
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
			bus.Send("ui", "notify", fmt.Sprintf("Trezor disconnected: %s", t.Name))
			trezors = append(trezors[:i], trezors[i+1:]...)
			ui.TopLeftFlow.RemovePane(t.Pane)
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
	bus.Send("ui", "notify", fmt.Sprintf("Trezor connected: %s", t.Name))
}

func disconnected(m *bus.B_UsbDisconnected) {
	remove(m.USB_ID)
}

func find_by_name(name []*cmn.Signer) *Trezor {
	trezors_mutex.Lock()
	defer trezors_mutex.Unlock()

	for _, t := range trezors {
		for _, n := range name {
			if t.Name == n.Name {
				return t
			}
		}
	}

	return nil
}

func provide_device(msg *bus.Message, sn string) *Trezor {
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

	pane := ui.NewAuxPane("HW Trezor", "<w><c>Please <blink>connect</blink> your Trezor device:\n\n <u><b>"+
		name+"</b></u>"+copies+"\n\n<button text:Cancel>")
	ui.TopLeftFlow.AddPane(pane)

	defer func() {
		ui.TopLeftFlow.RemovePane(pane)
	}()

	bus.TimerLoop(60, 3, msg.TimerID, func() (any, error, bool) {

		if !pane.On {
			return nil, fmt.Errorf("Canceled"), true
		}

		t = find_by_name(s_list)
		if t != nil {
			ui.TopLeftFlow.RemovePane(pane)
			return nil, nil, true
		}
		return nil, nil, false
	})

	return t
}

func (t *Trezor) isSkipPassword() bool {

	w := cmn.CurrentWallet
	if w == nil {
		log.Error().Msg("UsePassword: No wallet")
		return false
	}

	on, ok := w.ParamInt["trezor/"+t.Features.GetDeviceId()+"/skip_password"]
	if !ok {
		return false
	}

	return on == 1
}

func (t *Trezor) setSkipPassword(on bool) {
	w := cmn.CurrentWallet
	if w == nil {
		log.Error().Msg("UsePassword: No wallet")
		return
	}

	if on {
		w.ParamInt["trezor/"+t.Features.GetDeviceId()+"/skip_password"] = 1
	} else {
		delete(w.ParamInt, "trezor/"+t.Features.GetDeviceId()+"/skip_password")
	}
}
