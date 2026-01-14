package trezor

import (
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/hw/trezor/trezorproto"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

func get_addresses(msg *bus.Message) (*bus.B_SignerGetAddresses_Response, error) {
	w := cmn.CurrentWallet
	if w == nil {
		return nil, errors.New("no wallet")
	}

	data, ok := msg.Data.(*bus.B_SignerGetAddresses)
	if !ok {
		return nil, bus.ErrInvalidMessageData
	}

	rd := &bus.B_SignerGetAddresses_Response{}
	t := provide_device(msg, data.Name)
	if t == nil {
		return rd, fmt.Errorf("no device found with name %s", data.Name)
	}

	rd.Addresses = []common.Address{}
	rd.Paths = []string{}

	for i := 0; i < data.Count; i++ {
		path := fmt.Sprintf(data.Path, data.StartFrom+i)

		log.Debug().Msgf("GetAddresses: Getting address: %s", path)

		dp, err := accounts.ParseDerivationPath(path)
		if err != nil {
			log.Error().Err(err).Msgf("GetAddresses: Error parsing path: %s", path)
			return rd, err
		}

		eth_addr := new(trezorproto.EthereumAddress)
		if err := t.Call(
			msg,
			&trezorproto.EthereumGetAddress{AddressN: []uint32(dp)}, eth_addr); err != nil {
			log.Error().Err(err).Msgf("GetAddresses: Error getting address: %s", path)
			return rd, err
		}

		var a common.Address

		if addr := eth_addr.GetXOldAddress(); len(addr) > 0 { // Older firmwares use binary formats
			a = common.BytesToAddress(addr)
		}
		if addr := eth_addr.GetAddress(); len(addr) > 0 { // Newer firmwares use hexadecimal formats
			a = common.HexToAddress(addr)
		}

		rd.Addresses = append(rd.Addresses, a)
		rd.Paths = append(rd.Paths, path)
	}

	return rd, nil
}
