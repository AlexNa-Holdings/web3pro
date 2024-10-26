package trezor

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/hw/trezor/trezorproto"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

func (d *Trezor) Call(m *bus.Message, req proto.Message, result proto.Message) error {

	kind, reply, err := d.RawCall(m, req)
	if err != nil {
		log.Error().Msgf("Call: Error calling device: %s", err)
		return err
	}
	for {
		switch kind {
		case trezorproto.MessageType_MessageType_PinMatrixRequest:
			{
				log.Trace().Msg("*** Enter PIN ...")
				pr := m.Fetch(d.Pane.ViewName, "get_pin", nil)
				if pr.Error != nil {
					log.Error().Err(pr.Error).Msg("Error fetching pin")
					d.RawCall(m, &trezorproto.Cancel{})
					return pr.Error
				}

				pinStr := string(d.Pane.Pin)

				if len(pinStr) < 1 {
					log.Error().Msgf("Call: Invalid PIN provided")
					d.RawCall(m, &trezorproto.Cancel{})
					return errors.New("trezor: Invalid PIN provided")
				}

				for _, ch := range pinStr {
					if !strings.ContainsRune("123456789", ch) {
						log.Error().Msgf("Call: Invalid PIN provided")
						d.RawCall(m, &trezorproto.Cancel{})
						return errors.New("trezor: Invalid PIN provided")
					}
				}
				// send pin
				kind, reply, err = d.RawCall(m, &trezorproto.PinMatrixAck{Pin: &pinStr})
				if err != nil {
					log.Error().Msgf("Call: Error sending pin: %s", err)
					return err
				}

				if kind == trezorproto.MessageType_MessageType_Failure {
					return errors.New("trezor: " + "PIN request canceled")
				}

				log.Trace().Msgf("Trezor pin success. kind: %s\n", MessageName(kind))
			}
		case trezorproto.MessageType_MessageType_PassphraseRequest:
			{
				passStr := ""
				if !d.isSkipPassword() {

					pr := m.Fetch(d.Pane.ViewName, "get_pass", nil)
					if pr.Error != nil {
						log.Error().Err(pr.Error).Msg("Error fetching pass")
						d.RawCall(m, &trezorproto.Cancel{})
						return pr.Error
					}

					passStr = d.Pane.Pass
				}
				// send it
				kind, reply, err = d.RawCall(m, &trezorproto.PassphraseAck{Passphrase: &passStr})
				if err != nil {
					return err
				}

				if kind == trezorproto.MessageType_MessageType_Failure {
					return errors.New("trezor: " + "Passphrase request canceled")
				}

				log.Trace().Msgf("Trezor pass success. kind: %s\n", MessageName(kind))
			}
		case trezorproto.MessageType_MessageType_ButtonRequest:
			{
				log.Trace().Msg("*** NB! Button request on your Trezor screen ...")
				// Trezor is waiting for user confirmation, ack and wait for the next message
				kind, reply, err = d.RawCall(m, &trezorproto.ButtonAck{})
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
