package ledger

import (
	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/rs/zerolog/log"
)

func getName(msg *bus.Message, usb_id string) (string, error) {

	ledger := find_by_usb_id(usb_id)
	if ledger == nil {
		return "", nil
	}

	save_mode := ledger.Pane.Mode
	save_template := ledger.Pane.GetTemplate()
	defer func() {
		ledger.Pane.SetTemplate(save_template)
		ledger.Pane.SetMode(save_mode)
	}()

	ledger.Pane.SetTemplate("<w><c>\nPlease <blink>allow</blink> to read the Ledger name\n")
	ledger.Pane.SetMode("template")

	err := provide_eth_app(msg, usb_id, "BOLOS")
	if err != nil {
		return "", err
	}

	r, err := call(msg, usb_id, &GET_DEVICE_NAME_APDU, nil)
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error getting device name: %s", usb_id)
		return "", err
	}

	log.Trace().Msgf("Initialized ledger dev: %v\n", string(r))

	return string(r), nil
}
