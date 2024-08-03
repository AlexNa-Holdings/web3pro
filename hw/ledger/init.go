package ledger

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

var CLEANING_APDU = APDU{0xe0, 0x50, 0x00, 0x00, nil}
var GET_DEVICE_NAME_APDU = APDU{0xe0, 0xd2, 0x00, 0x00, nil}
var GET_INFO_APDU = APDU{0xe0, 0x01, 0x00, 0x00, nil}

func init_ledger(usb_id string) (*Ledger, error) {
	t := &Ledger{
		USB_ID: usb_id,
	}

	r, err := rawCall(usb_id, &CLEANING_APDU)
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error cleaning device: %s", usb_id)
		return nil, err
	}
	log.Debug().Msgf("Cleaning Received: '%s'", hexutil.Bytes(r))

	r, err = rawCall(usb_id, &GET_INFO_APDU)
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error getting device info: %s", usb_id)
		return nil, err
	}
	log.Debug().Msgf("Info Received: '%s'", hexutil.Bytes(r))

	r, err = rawCall(usb_id, &GET_DEVICE_NAME_APDU)
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error getting device name: %s", usb_id)
		return nil, err
	}

	log.Debug().Msgf("Name Received: '%s'", hexutil.Bytes(r))

	t.Name = string(r) //PARSE

	log.Trace().Msgf("Initialized ledger dev: %v\n", t.Name)

	return t, nil
}
