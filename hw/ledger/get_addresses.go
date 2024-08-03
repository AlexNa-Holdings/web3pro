package ledger

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/ethereum/go-ethereum/common"
)

func get_addresses(m *bus.B_SignerGetAddresses) (*bus.B_SignerGetAddresses_Response, error) {
	rd := &bus.B_SignerGetAddresses_Response{}
	t := provide_device(m.Name)
	if t == nil {
		return rd, fmt.Errorf("no device found with name %s", m.Name)
	}

	rd.Addresses = []common.Address{}
	rd.Paths = []string{}

	return rd, nil
}
