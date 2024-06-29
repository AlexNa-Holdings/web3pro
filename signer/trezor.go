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
	//TODO
	return false
}

func (d TrezorDriver) GetAddresses(s *Signer, path_format string, start_from int, count int) ([]address.Address, error) {
	return []address.Address{}, nil // TODO
}
