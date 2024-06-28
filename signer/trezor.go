package signer

import (
	"github.com/AlexNa-Holdings/web3pro/address"
)

type TrezorDriver struct {
}

func NewTrezorDriver() TrezorDriver {
	return TrezorDriver{}
}

func (d *TrezorDriver) Open() error {
	return nil
}

func (d TrezorDriver) IsConnected(s *Signer) bool {
	// devices, err := core.List()
	// if err != nil {
	// 	log.Error().Msgf("Error listing USB devices", "err", err)
	// 	return false
	// }

	// sns := []string{d.SN}
	// for _, c := range d.Copies {
	// 	sns = append(sns, c.SN)
	// }

	// for _, device := range devices {
	// 	if device.Product == "TREZOR" && cmn.IsInArray(sns, device.Serial) {
	// 		return true
	// 	}
	// }
	return false
}

func (d TrezorDriver) GetAddresses(s *Signer, path_format string, start_from int, count int) ([]address.Address, error) {
	return []address.Address{}, nil // TODO
}
