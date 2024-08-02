package trezor

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/hw/trezor/trezorproto"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

func (d *Trezor) Call(req proto.Message, result proto.Message) error {

	kind, reply, err := d.RawCall(req)
	if err != nil {
		log.Error().Msgf("Call: Error calling device: %s", err)
		return err
	}
	for {
		switch kind {
		case trezorproto.MessageType_MessageType_PinMatrixRequest:
			{
				log.Trace().Msg("*** Enter PIN ...")
				pin, err := d.RequsetPin()
				if err != nil {
					log.Error().Msgf("Call: Error getting pin: %s", err)
					d.RawCall(&trezorproto.Cancel{})
					return err
				}

				pinStr := string(pin)
				for _, ch := range pinStr {
					if !strings.ContainsRune("123456789", ch) || len(pin) < 1 {
						log.Error().Msgf("Call: Invalid PIN provided")
						d.RawCall(&trezorproto.Cancel{})
						return errors.New("trezor: Invalid PIN provided")
					}
				}
				// send pin
				kind, reply, err = d.RawCall(&trezorproto.PinMatrixAck{Pin: &pinStr})
				if err != nil {
					log.Error().Msgf("Call: Error sending pin: %s", err)
					return err
				}
				log.Trace().Msgf("Trezor pin success. kind: %s\n", MessageName(kind))
			}
		case trezorproto.MessageType_MessageType_PassphraseRequest:
			{
				log.Trace().Msg("Enter Pass	phrase")
				pass, err := d.RequsetPassword()
				if err != nil {
					d.RawCall(&trezorproto.Cancel{})
					return err
				}
				passStr := string(pass)
				// send it
				kind, reply, err = d.RawCall(&trezorproto.PassphraseAck{Passphrase: &passStr})
				if err != nil {
					return err
				}
				log.Trace().Msgf("Trezor pass success. kind: %s\n", MessageName(kind))
			}
		case trezorproto.MessageType_MessageType_ButtonRequest:
			{
				log.Trace().Msg("*** NB! Button request on your Trezor screen ...")
				// Trezor is waiting for user confirmation, ack and wait for the next message
				kind, reply, err = d.RawCall(&trezorproto.ButtonAck{})
				if err != nil {
					return err
				}
				log.Trace().Msgf("Trezor button success. kind: %s\n", MessageName(kind))
			}
		case trezorproto.MessageType_MessageType_Failure:
			{
				// Trezor returned a failure, extract and return the message
				failure := new(trezorproto.Failure)
				if err := proto.Unmarshal(reply, failure); err != nil {
					return err
				}
				// fmt.Printf("Trezor failure success. kind: %s\n", MessageName(kind))
				return errors.New("trezor: " + failure.GetMessage())
			}
		default:
			{
				resultKind := MessageType(result)
				if resultKind != kind {
					return fmt.Errorf("trezor: expected reply type %s, got %s", MessageName(resultKind), MessageName(kind))
				}
				return proto.Unmarshal(reply, result)
			}
		}
	}
}

func (d *Trezor) RequsetPin() (string, error) {
	template := "<c><w>\n<l id:pin text:'____________'> <button text:'\U000f006e ' id:back>\n\n"

	ids := []int{7, 8, 9, 4, 5, 6, 1, 2, 3}

	for i := 0; i < 9; i++ {
		template += fmt.Sprintf("<button color:g.HelpFgColor bgcolor:g.HelpBgColor text:' - ' id:%d> ", ids[i])
		if (i+1)%3 == 0 {
			template += "\n\n"
		}
	}
	template += "<button text:OK> <button text:Cancel>"
	pin := ""

	bus.Fetch("ui", "hail", &bus.B_Hail{
		Title:    "Enter Trezor PIN",
		Template: template,
		OnClickHotspot: func(h *bus.B_Hail, v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				s := cmn.Split(hs.Value)
				command, value := s[0], s[1]

				switch command {
				case "button":
					switch value {
					case "back":
						if len(pin) > 0 {
							pin = pin[:len(pin)-1]
							v.GetHotspotById("pin").SetText(strings.Repeat("*", len(pin)) + "______________")
						}
					case "1", "2", "3", "4", "5", "6", "7", "8", "9":
						pin += value
						v.GetHotspotById("pin").SetText(strings.Repeat("*", len(pin)) + "______________")
					}
				}
			}
		},
	})

	log.Debug().Msgf("PIN: %s", pin)

	if pin == "" {
		return "", errors.New("pin request canceled")
	}

	return pin, nil

}

func (d *Trezor) RequsetPassword() (string, error) {
	password := ""
	canceled := false

	bus.Fetch("ui", "hail", &bus.B_Hail{
		Title: "Select Wallet Type",
		Template: `<c><w>
<button text:Standard color:g.HelpFgColor bgcolor:g.HelpBgColor id:standard> <button text:Hidden color:g.HelpFgColor bgcolor:g.HelpBgColor id:hidden> 

<button text:Cancel>`,

		OnClickHotspot: func(h *bus.B_Hail, v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				s := cmn.Split(hs.Value)
				command, value := s[0], s[1]

				switch command {
				case "button":
					switch value {
					case "standard":
						bus.Send("ui", "remove_hail", h)
					case "hidden":
						h.TimerPaused = true
						v.GetGui().ShowPopup(&gocui.Popup{
							Title: "Enter Trezor Password",
							Template: `<c><w>
Password: <input id:password size:16 masked:true>

<button text:OK> <button text:Cancel>`,
							OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
								if hs != nil {
									switch hs.Value {
									case "button OK":
										password = v.GetInput("password")
										v.GetGui().HidePopup()
										bus.Send("ui", "remove_hail", h)
									case "button Cancel":
										v.GetGui().HidePopup()
									}
								}
							},
							OnClose: func(v *gocui.View) {
								h.TimerPaused = false
							},
						})
					}
				}
			}
		},
		OnCancel: func(h *bus.B_Hail) {
			canceled = true
		},
	})

	if canceled {
		return "", errors.New("password request canceled")
	}

	return password, nil
}
