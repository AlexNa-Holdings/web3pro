package trezor

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/hw/trezor/trezorproto"
	"github.com/ava-labs/coreth/accounts"
	"github.com/ethereum/go-ethereum/common"
)

func get_addresses(m *bus.B_HwGetAddresses) (*bus.Message, error) {
	rd := &bus.B_HwGetAddresses_Response{}
	r := &bus.Message{Data: rd}
	t := provide_device(m.Name)
	if t == nil {
		return r, fmt.Errorf("No device found with name %s", m.Name)
	}

	rd.Addresses = []common.Address{}

	for i := 0; i < m.Count; i++ {
		path := fmt.Sprintf(m.Path, m.StartFrom+i)
		dp, err := accounts.ParseDerivationPath(path)
		if err != nil {
			return r, err
		}

		eth_addr := new(trezorproto.EthereumAddress)
		if err := t.Call(
			&trezorproto.EthereumGetAddress{AddressN: []uint32(dp)}, eth_addr); err != nil {
			return r, err
		}

		var a common.Address

		if addr := eth_addr.GetXOldAddress(); len(addr) > 0 { // Older firmwares use binary formats
			a = common.BytesToAddress(addr)
		}
		if addr := eth_addr.GetAddress(); len(addr) > 0 { // Newer firmwares use hexadecimal formats
			a = common.HexToAddress(addr)
		}

		rd.Addresses = append(rd.Addresses, a)
	}
	return r, nil
}
