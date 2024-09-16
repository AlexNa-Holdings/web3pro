package ledger

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/ava-labs/coreth/accounts"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/rs/zerolog/log"
)

func signTypedData_v4(msg *bus.Message) (string, error) {
	m, _ := msg.Data.(*bus.B_SignerSignTypedData_v4)

	ledger := provide_device(m.Name)
	if ledger == nil {
		return "", fmt.Errorf("no device found with name %s", m.Name)
	}

	err := provide_eth_app(ledger.USB_ID, "Ethereum")
	if err != nil {
		return "", err
	}

	var payload []byte

	dp, err := accounts.ParseDerivationPath(m.Path)
	if err != nil {
		log.Error().Err(err).Msgf("SignTypedData: Error parsing path: %s", m.Path)
		return "", err
	}
	payload = append(payload, serializePath(dp)...)

	_, data, err := apitypes.TypedDataAndHash(m.TypedData)
	if err != nil {
		log.Error().Msgf("SignTypedData: Failed to hash typed data: %v", err)
		return "", err
	}

	payload = append(payload, data[2:34]...)
	payload = append(payload, data[34:66]...)

	reply, err := call(ledger.USB_ID, &GET_SIGN_MSG_APDU, payload, generalHail, 5)
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error getting device name: %s", ledger.USB_ID)
		return "", err
	}

	var sig []byte
	sig = append(sig, reply[1:]...) // R + S
	sig = append(sig, reply[0])     // V

	log.Debug().Msgf("Signature: 0x%x", sig)
	log.Debug().Msgf("Len: %d", len(sig))

	if len(sig) == 65 && (sig[64] == 0 || sig[64] == 1) {
		sig[64] += 27
	}

	return fmt.Sprintf("0x%x", sig), nil
}
