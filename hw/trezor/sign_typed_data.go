package trezor

import (
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/hw/trezor/trezorproto"
	"github.com/ava-labs/coreth/accounts"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/rs/zerolog/log"
)

func signTypedData_v4(msg *bus.Message) (string, error) {
	m, _ := msg.Data.(*bus.B_SignerSignTypedData_v4)

	t := provide_device(m.Name)
	if t == nil {
		return "", fmt.Errorf("Trezor not found: %s", m.Name)
	}

	data, _, err := apitypes.TypedDataAndHash(m.TypedData)
	if err != nil {
		log.Error().Msgf("SignTypedData: Failed to hash typed data: %v", err)
		return "", err
	}

	dp, err := accounts.ParseDerivationPath(m.Path)
	if err != nil {
		log.Error().Err(err).Msgf("SignTypedData: Error parsing path: %s", m.Path)
		return "", err
	}

	sig := new(trezorproto.EthereumMessageSignature)

	if err := t.Call(
		msg,
		&trezorproto.EthereumSignMessage{
			AddressN: dp,
			Message:  data,
		}, sig); err != nil {
		log.Error().Err(err).Msgf("SignTypedData: Error signing typed data: %s", m.Path)

		return "", err
	}

	log.Debug().Msgf("Signature: 0x%x", sig.GetSignature())
	log.Debug().Msgf("Address: %s", sig.GetAddress())

	sig_bytes := sig.GetSignature()

	log.Debug().Msgf("Len: %d", len(sig_bytes))

	if len(sig_bytes) == 65 && (sig_bytes[64] == 0 || sig_bytes[64] == 1) {
		sig_bytes[64] += 27
	}

	if sig.GetAddress() != m.Address.Hex() {
		return "", errors.New("wrong Trezor device")
	}

	return fmt.Sprintf("0x%x", sig.GetSignature()), nil
}
