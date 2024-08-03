package ledger

import (
	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/rs/zerolog/log"
)

var CLEANING_APDU = APDU{0xe0, 0x50, 0x00, 0x00}
var GET_DEVICE_NAME_APDU = APDU{0xe0, 0xd2, 0x00, 0x00}
var GET_INFO_APDU = APDU{0xe0, 0x01, 0x00, 0x00}
var GET_VERSION_APDU = APDU{0xe0, 0x01, 0x00, 0x00}
var BACKUP_APP_STORAGE = APDU{0xe0, 0x6b, 0x00, 0x00}
var GET_APP_STORAGE_INFO = APDU{0xe0, 0x6a, 0x00, 0x00}
var RESTORE_APP_STORAGE = APDU{0xe0, 0x6d, 0x00, 0x00}
var RESTORE_APP_STORAGE_COMMIT = APDU{0xe0, 0x6e, 0x00, 0x00}
var RESTORE_APP_STORAGE_INIT = APDU{0xe0, 0x6c, 0x00, 0x00}

// GET_VERSION: 0x02,
// GET_ADDRESS: 0x03,
// SET_ADDRESS: 0x05,
// PROVIDE_ESDT_INFO: 0x08,

var getNameHail = &bus.B_Hail{
	Title: "Get Ledger Name",
	Template: `<c><w>
Please unlock your Ledger device and allow to read its name

<button text:Cancel>`,
}

func getName(usb_id string) (string, error) {
	r, err := call(usb_id, &GET_DEVICE_NAME_APDU, nil, getNameHail, 0)
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error getting device name: %s", usb_id)
		return "", err
	}

	log.Trace().Msgf("Initialized ledger dev: %v\n", string(r))

	return string(r), nil
}
