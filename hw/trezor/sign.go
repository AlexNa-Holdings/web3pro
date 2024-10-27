package trezor

import (
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/hw/trezor/trezorproto"
	"github.com/ava-labs/coreth/accounts"
	"github.com/rs/zerolog/log"
)

func sign(msg *bus.Message) (string, error) {
	w := cmn.CurrentWallet
	if w == nil {
		return "", errors.New("no wallet")
	}

	m, _ := msg.Data.(*bus.B_SignerSign)

	t := provide_device(m.Name)
	if t == nil {
		return "", fmt.Errorf("Trezor not found: %s", m.Name)
	}

	dp, err := accounts.ParseDerivationPath(m.Path)
	if err != nil {
		log.Error().Err(err).Msgf("SignTypedData: Error parsing path: %s", m.Path)
		return "", err
	}

	request := &trezorproto.EthereumSignMessage{
		AddressN: dp,
		Message:  m.Data,
	}

	response := new(trezorproto.EthereumMessageSignature)

	save_mode := t.Pane.Mode
	save_template := t.Pane.GetTemplate()
	defer func() {
		t.Pane.SetTemplate(save_template)
		t.Pane.SetMode(save_mode)
	}()

	t.Pane.SetTemplate("<w><c>\n<blink>" + cmn.ICON_ALERT + "</blink>Please review tand sign it with your Trezor device\n")
	t.Pane.SetMode("template")

	if err := t.Call(msg, request, response); err != nil {
		log.Error().Err(err).Msgf("Sign: Error signing typed data(1): %s", m.Path)
		return "", err
	}

	log.Debug().Msgf("Signature: 0x%x", response.GetSignature())
	log.Debug().Msgf("Address: %s", response.GetAddress())

	sig_bytes := response.GetSignature()

	log.Debug().Msgf("Len: %d", len(sig_bytes))

	if len(sig_bytes) == 65 && (sig_bytes[64] == 0 || sig_bytes[64] == 1) {
		sig_bytes[64] += 27
	}

	if response.GetAddress() != m.Address.Hex() {
		return "", errors.New("wrong Trezor device")
	}

	return fmt.Sprintf("0x%x", response.GetSignature()), nil
}
