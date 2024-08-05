package ledger

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/rs/zerolog/log"
)

func call(usb_id string, apdu *APDU, data []byte, hail *bus.B_Hail, hail_delay int) ([]byte, error) {
	var err error

	r, err := rawCall(usb_id, apdu, data, hail, hail_delay)

	for {
		switch {
		case err == nil:
			return r, nil
		case strings.Contains(err.Error(), "LOCKED_DEVICE"):
			bus.Fetch("ui", "hail", &bus.B_Hail{
				Title: "Unlock Ledger",
				Template: `<c><w>
Please unlock your Ledger device

<button text:Cancel>`,
				OnTick: func(h *bus.B_Hail, tick int) {
					if tick%3 == 0 {
						r, err = rawCall(usb_id, apdu, data, hail, hail_delay)
						if err == nil || !strings.Contains(err.Error(), "LOCKED_DEVICE") {
							bus.Send("ui", "remove-hail", h)
						}
					}
				},
			})
		default:
			log.Error().Err(err).Msg("Error calling ledger")
			return nil, err
		}
	}

}
