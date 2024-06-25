package signer

import (
	"errors"

	"github.com/AlexNa-Holdings/web3pro/address"
	"github.com/AlexNa-Holdings/web3pro/usb_support"
	"github.com/karalabe/usb"
	"github.com/rs/zerolog/log"
)

type TrezorDriver struct {
	*Signer
	usb.Device
}

func NewTrezorDriver(s *Signer) (TrezorDriver, error) {

	if s.Type != "trezor" {
		return TrezorDriver{}, errors.New("invalid signer type")
	}

	return TrezorDriver{
		Signer: s,
	}, nil
}

func (d TrezorDriver) IsConnected() bool {

	devices, err := usb_support.List()
	if err != nil {
		log.Error().Msgf("Error listing USB devices", "err", err)
		return false
	}

	for _, device := range devices {
		if device.Product == "TREZOR" && device.Serial == d.SN {
			return true
		}
	}
	return false
}

func (d TrezorDriver) GetAddresses(path_format string, start_from int, count int) ([]address.Address, error) {
	addresses := []address.Address{}

	// find sutable device

	if d.IsConnected() {
		l, err := usb_support.List()
		if err != nil {
			log.Error().Msgf("Error listing USB devices: %v", err)
			return addresses, err
		}

		for _, info := range l {
			if info.Product == "TREZOR" && info.Serial == d.SN {
				d.Device, err = info.Open()
				if err != nil {
					log.Error().Msgf("Error opening USB device: %v", err)
					return addresses, err
				}
			}
		}

		// p := trezor.Initialize

		log.Debug().Msgf("Device: %v", d.Device)
	}

	return addresses, nil

}
