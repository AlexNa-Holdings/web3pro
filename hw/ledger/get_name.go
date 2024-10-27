package ledger

import (
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/rs/zerolog/log"
)

func getName(usb_id string) (string, error) {

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

	ledger.Pane.SetTemplate("<w><c>\n<blink>" + cmn.ICON_ALERT + "</blink>Please unlock your Ledger device and allow to read its name\n")
	ledger.Pane.SetMode("template")

	err := provide_eth_app(usb_id, "BOLOS")
	if err != nil {
		return "", err
	}

	r, err := call(usb_id, &GET_DEVICE_NAME_APDU, nil)
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error getting device name: %s", usb_id)
		return "", err
	}

	log.Trace().Msgf("Initialized ledger dev: %v\n", string(r))

	return string(r), nil
}
