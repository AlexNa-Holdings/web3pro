package cmn

import (
	"errors"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/mnemonics"
	"github.com/AlexNa-Holdings/web3pro/usb"
	"github.com/ethereum/go-ethereum/common"

	"github.com/rs/zerolog/log"
)

type Signer struct {
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	SN     string   `json:"sn"`
	Copies []string `json:"copies"`
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

var KNOWN_SIGNER_TYPES = []string{"trezor", "ledger", "mnemonics"}

func GetDeviceType(vid int, pid int) string {

	if usb.IsTrezor(uint16(vid), uint16(pid)) {
		return "trezor"
	}

	if usb.IsLedger(uint16(vid), uint16(pid)) {
		return "ledger"
	}
	return ""
}

func (s *Signer) GetDriver() (SignerDriver, error) {
	switch s.Type {
	case "trezor":
		return WalletTrezorDriver, nil
	case "mnemonics":
		return WalletMnemonicsDriver, nil
	}

	return nil, errors.New("unknown signer type")
}

func GetDeviceName(e usb.EnumerateEntry) (string, error) {
	log.Trace().Msgf("GetDeviceName: %x %x", e.Vendor, e.Product)
	t := GetDeviceType(e.Vendor, e.Product)
	switch t {
	case "trezor":
		return "TODO", nil //WalletTrezorDriver.GetName(e.Path)
	case "ledger":
		return "Ledger ID", nil //TODO
	}
	return "", errors.New("unknown signer type")

}

func (s *Signer) GetAddresses(path string, start_from int, count int) ([]common.Address, []string, error) {
	if s.Type == "mnemonics" {
		m, err := mnemonics.NewFromSN(s.SN)
		if err != nil {
			log.Error().Err(err).Msgf("GetAddresses: Error getting addresses: %s (%s)", s.Name, s.Type)
			return []common.Address{}, []string{}, err
		}

		addresses, paths, err := m.GetAddresses(path, start_from, count)
		if err != nil {
			log.Error().Err(err).Msgf("GetAddresses: Error getting addresses: %s (%s)", s.Name, s.Type)
			return []common.Address{}, []string{}, err
		}
		return addresses, paths, nil
	}

	m := bus.Fetch("hw", "get-addresses", &bus.B_HwGetAddresses{
		Type:      s.Type,
		Name:      s.GetFamilyNames(),
		Path:      path,
		StartFrom: start_from,
		Count:     count})
	if m.Error != nil {
		log.Error().Err(m.Error).Msgf("GetAddresses: Error getting addresses: %s (%s)", s.Name, s.Type)
		return []common.Address{}, []string{}, m.Error
	}

	r, ok := m.Data.(*bus.B_HwGetAddresses_Response)
	if !ok {
		log.Error().Msgf("GetAddresses: Error getting addresses: %s (%s)", s.Name, s.Type)
		return []common.Address{}, []string{}, errors.New("error getting addresses")
	}

	return r.Addresses, r.Paths, nil
}

func (s *Signer) IsConnected() bool {

	if s.Type == "mnemonics" {
		return true
	}

	r := bus.Fetch("hw", "is-connected", &bus.B_HwIsConnected{Type: s.Type, Name: s.GetFamilyNames()})
	if r.Error != nil {
		log.Error().Err(r.Error).Msgf("Error checking connection: %s (%s)", s.Name, s.Type)
		return false
	}

	m, ok := r.Data.(*bus.B_HwIsConnected_Response)
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
