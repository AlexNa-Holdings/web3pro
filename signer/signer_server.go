package signer

import (
	"errors"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/rs/zerolog/log"
)

var trezorDriver = NewTrezorDriver()
var mnemonicsDriver = NewMnemonicDriver()
var ledgerDriver = NewLedgerDriver()

func Init() {
	go Loop()
}

func Loop() {
	ch := bus.Subscribe("signer")
	for {
		select {
		case msg := <-ch:
			if msg.RespondTo != 0 {
				continue // ignore responses
			}

			switch msg.Type {
			case "init":
				req, ok := msg.Data.(bus.B_SignerInit)
				if !ok {
					log.Error().Msg("Invalid message data")
					msg.Respond(nil, bus.ErrInvalidMessageData)
					continue
				}

				name, params, err := InitSigner(req.USB_ID, req.Type)
				if err != nil {
					log.Error().Err(err).Msg("Error getting name")
					msg.Respond(nil, err)
					continue
				}
				msg.Respond(&bus.B_SignerInit_Response{
					Name:      name,
					HW_Params: params,
				}, nil)
			}
		}
	}
}

func InitSigner(id string, t string) (string, any, error) {
	switch t {
	case "trezor":
		return trezorDriver.InitSigner(id)
	case "ledger":
		//	return ledgerDriver.InitSigner(id)
	}
	return "", nil, errors.New("unknown hardware wallet type")
}
