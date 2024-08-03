package ledger

import "github.com/rs/zerolog/log"

func init_ledger(usb_id string) (*Ledger, error) {
	t := &Ledger{
		USB_ID: usb_id,
	}

	_, err := rawCall(usb_id, &CLEANING_APDU)
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error cleaning device: %s", usb_id)
		return nil, err
	}

	r, err := rawCall(usb_id, &GET_DEVICE_NAME_APDU)
	if err != nil {
		log.Error().Err(err).Msgf("Init: Error getting device name: %s", usb_id)
		return nil, err
	}

	t.Name = string(r) //PARSE

	log.Trace().Msgf("Initialized ledger dev: %v\n", t.Name)

	return t, nil
}
