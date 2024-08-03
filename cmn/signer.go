package cmn

import (
	"errors"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/ethereum/go-ethereum/common"

	"github.com/rs/zerolog/log"
)

type Signer struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	MasterKey string   `json:"master-key"`
	Copies    []string `json:"copies"`
}

var STANDARD_DERIVATIONS = map[string]struct {
	Name   string
	Format string
}{
	"legacy": {
		Name:   "Legacy (MEW, MyCrypto) m/44'/60'/0'/0/%d",
		Format: "m/44'/60'/0'/0/%d",
	},
	"ledger-live": {
		Name:   "Ledger Live m/44'/60'/%d'/0/0",
		Format: "m/44'/60'/%d'/0/0",
	},
	"default": {
		Name:   "Default m/44'/60'/0'/0/%d",
		Format: "m/44'/60'/0'/0/%d",
	},
}

func (s *Signer) GetAddresses(path string, start_from int, count int) ([]common.Address, []string, error) {
	m := bus.Fetch("signer", "get-addresses", &bus.B_SignerGetAddresses{
		Type:      s.Type,
		Name:      s.GetFamilyNames(),
		Path:      path,
		MasterKey: s.MasterKey,
		StartFrom: start_from,
		Count:     count})

	if m.Error != nil {
		log.Error().Err(m.Error).Msgf("GetAddresses: Error getting addresses: %s (%s) err: %s", s.Name, s.Type, m.Error)
		return []common.Address{}, []string{}, m.Error
	}

	r, ok := m.Data.(*bus.B_SignerGetAddresses_Response)
	if !ok {
		log.Error().Msgf("GetAddresses: Error getting addresses: %s (%v)", s.Name, r)
		return []common.Address{}, []string{}, errors.New("error getting addresses")
	}

	return r.Addresses, r.Paths, nil
}

func (s *Signer) IsConnected() bool {
	r := bus.Fetch("signer", "is-connected", &bus.B_SignerIsConnected{Type: s.Type, Name: s.GetFamilyNames()})
	if r.Error != nil {
		log.Error().Err(r.Error).Msgf("Error checking connection: %s (%s)", s.Name, s.Type)
		return false
	}

	m, ok := r.Data.(*bus.B_SignerIsConnected_Response)
	if !ok {
		log.Error().Msgf("Error checking connection: %s (%s)", s.Name, s.Type)
		return false
	}

	return m.Connected
}

func (s *Signer) GetFamilyNames() []string {
	r := []string{s.Name}
	r = append(r, s.Copies...)
	return r
}
