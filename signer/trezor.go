package signer

import (
	"errors"

	"github.com/AlexNa-Holdings/web3pro/usb"
	"github.com/ethereum/go-ethereum/log"
)

type TrezorDriver struct {
	*Signer
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

	devices, err := usb.List()
	if err != nil {
		log.Error("Error listing USB devices", "err", err)
		return false
	}

	for _, device := range devices {
		if device.Product == "TREZOR" && device.Serial == d.SN {
			return true
		}
	}
	return false
}
