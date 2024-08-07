package ledger

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/ava-labs/coreth/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

func get_addresses(m *bus.B_SignerGetAddresses) (*bus.B_SignerGetAddresses_Response, error) {
	rd := &bus.B_SignerGetAddresses_Response{}
	ledger := provide_device(m.Name)
	if ledger == nil {
		return rd, fmt.Errorf("no device found with name %s", m.Name)
	}

	err := provide_eth_app(ledger.USB_ID, "Ethereum")
	if err != nil {
		return rd, err
	}

	rd.Addresses = []common.Address{}
	rd.Paths = []string{}

	for i := 0; i < m.Count; i++ {
		path := fmt.Sprintf(m.Path, m.StartFrom+i)

		bipPath, err := accounts.ParseDerivationPath(path)
		if err != nil {
			return rd, fmt.Errorf("error parsing path: %s", err)
		}

		data := serializePath(bipPath)

		r, err := call(ledger.USB_ID, &GET_ADDRESS_APDU, data, generalHail, 5)
		if err != nil {
			log.Error().Err(err).Msgf("Init: Error getting device name: %s", ledger.USB_ID)
			return rd, err
		}

		_, addr, err := parseGetAddressResponse(r)
		if err != nil {
			return rd, fmt.Errorf("error parsing get address response: %s", err)
		}

		rd.Addresses = append(rd.Addresses, addr)
		rd.Paths = append(rd.Paths, path)
	}

	return rd, nil
}

func parseGetAddressResponse(r []byte) (string, common.Address, error) {
	if len(r) < 66 {
		return "", common.Address{}, fmt.Errorf("empty response")
	}

	pubKey := hexutil.Encode(r[1:65])
	r = r[1+int(r[0]):]
	hexstr := r[1 : 1+int(r[0])]

	var addr common.Address
	if _, err := hex.Decode(addr[:], hexstr); err != nil {
		return pubKey, common.Address{}, err
	}
	return pubKey, addr, nil
}

func serializePath(path accounts.DerivationPath) []byte {
	buf := make([]byte, 1+len(path)*4)
	buf[0] = byte(len(path)) // First byte is the length of the path

	for i, v := range path {
		binary.BigEndian.PutUint32(buf[1+i*4:], v)
	}

	return buf
}
