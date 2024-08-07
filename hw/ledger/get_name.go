package ledger

import (
	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/rs/zerolog/log"
)

var getNameHail = &bus.B_Hail{
	Title: "Get Ledger Name",
	Template: `<c><w>
Please unlock your Ledger device and allow to read its name

<button text:Cancel>`,
}

func getName(usb_id string) (string, error) {

	err := provide_eth_app(usb_id, "BOLOS")
	if err != nil {
		return "", err
	}

	r, err := call(usb_id, &GET_DEVICE_NAME_APDU, nil, getNameHail, 0)
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error getting device name: %s", usb_id)
		return "", err
	}

	log.Trace().Msgf("Initialized ledger dev: %v\n", string(r))

	return string(r), nil
}
