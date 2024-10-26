package ledger

import (
	"encoding/binary"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ava-labs/coreth/accounts"
	"github.com/rs/zerolog/log"
)

func sign(msg *bus.Message) (string, error) {
	m, _ := msg.Data.(*bus.B_SignerSign)

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
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, uint32(len(m.Data)))
	payload = append(payload, lengthBytes...)
	payload = append(payload, m.Data...)

	log.Trace().Msgf("LEDGER: SIGN MESSAGE: PATH %s", m.Path)

	save_mode := ledger.Pane.Mode
	save_template := ledger.Pane.GetTemplate()
	defer func() {
		ledger.Pane.SetTemplate(save_template)
		ledger.Pane.SetMode(save_mode)
	}()

	ledger.Pane.SetTemplate("<w><c>\n<blink>" + gocui.ICON_ALERT + "</blink>Please sign the message on your device\n")
	ledger.Pane.SetMode("template")

	reply, err := call(ledger.USB_ID, &SIGN_MSG_PERSONAL_APDU, payload)
	if err != nil {
		log.Error().Err(err).Msgf("SignTypedData: Error signing typed data: %s", m.Path)
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
