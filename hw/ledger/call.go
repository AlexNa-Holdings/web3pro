package ledger

import (
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/rs/zerolog/log"
)

var generalTemplate = "<c><w>\nPlease <blink>make sure</blink> your Ledger device is connected and unlocked\n"

func call(msg *bus.Message, usb_id string, apdu *APDU, data []byte) ([]byte, error) {
	var err error

	ledger := find_by_usb_id(usb_id)
	if ledger == nil {
		return nil, fmt.Errorf("device %s not found", usb_id)
	}

	r, err := rawCall(msg, usb_id, apdu, data)

	for {
		switch {
		case err == nil:
			return r, nil
		case strings.Contains(err.Error(), "LOCKED_DEVICE"):

			save_mode := ledger.Pane.Mode
			save_template := ledger.Pane.GetTemplate()

			ledger.Pane.SetTemplate("<w><c>\nPlease <blink>unlock</blink> your Ledger device\n")
			ledger.Pane.SetMode("template")

			tl_data, err := bus.TimerLoop(60*2, 3, 0, func() (any, error, bool) {
				r, err = rawCall(msg, usb_id, apdu, data)
				if err == nil || !strings.Contains(err.Error(), "LOCKED_DEVICE") {
					return data, nil, true
				}
				return nil, nil, false
			})

			ledger.Pane.SetTemplate(save_template)
			ledger.Pane.SetMode(save_mode)

			if err != nil {
				return nil, err
			}

			var ok bool
			data, ok = tl_data.([]byte)
			if !ok {
				return nil, fmt.Errorf("error converting data")
			}

		case strings.Contains(err.Error(), "WRONG APP"):
			save_mode := ledger.Pane.Mode
			save_template := ledger.Pane.GetTemplate()

			ledger.Pane.SetTemplate("<w><c>\nPlease <blink>open</blink> Ethereum app on the device\n")
			ledger.Pane.SetMode("template")

			tl_data, err := bus.TimerLoop(60*2, 3, 0, func() (any, error, bool) {
				r, err = rawCall(msg, usb_id, apdu, data)
				if err == nil || !strings.Contains(err.Error(), "WRONG APP") {
					return data, nil, true
				}
				return nil, nil, false
			})

			ledger.Pane.SetTemplate(save_template)
			ledger.Pane.SetMode(save_mode)

			if err != nil {
				return nil, err
			}

			var ok bool
			data, ok = tl_data.([]byte)
			if !ok {
				return nil, fmt.Errorf("error converting data")
			}

		default:
			log.Error().Err(err).Msg("Error calling ledger")
			return nil, err
		}
	}

}
