package signer

import (
	"crypto/ecdsa"
	"errors"

	"github.com/AlexNa-Holdings/web3pro/address"
	"github.com/ava-labs/coreth/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog/log"
	"github.com/tyler-smith/go-bip32"
)

type SignerDriver interface {
	IsConnected() bool
	GetAddresses(path string, start_from int, count int) ([]address.Address, error)
}

type SignerCopy struct {
	Name string
	SN   string
}

type Signer struct {
	Name   string       `json:"name"`
	Type   string       `json:"type"`
	SN     string       `json:"sn"`
	Copies []SignerCopy `json:"copies"`
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

func GetType(manufacturer string, product string) string {
	if product == "TREZOR" {
		return "trezor"
	}

	if manufacturer == "Ledger" {
		return "ledger"
	}

	return ""
}

func (s *Signer) GetDriver() (SignerDriver, error) {
	switch s.Type {
	case "trezor":
		return NewTrezorDriver(s)
	// case "ledger":
	// 	return NewLedgerDriver(s)
	case "mnemonics":
		return NewMnemonicDriver(s)
	}

	return nil, errors.New("unknown signer type")
}

func (s *Signer) GetAddresses(path string, start_from int, count int) ([]address.Address, error) {

	driver, err := s.GetDriver()
	if err != nil {
		return []address.Address{}, err
	}

	return driver.GetAddresses(path, start_from, count)

}

func (s *Signer) IsConnected() bool {
	driver, err := s.GetDriver()
	if err != nil {
		log.Error().Err(err).Msgf("Error getting driver: %s (%s)", s.Name, s.Type)
		return false
	}

	return driver.IsConnected()
}

// deriveKey derives a key from the master key using the specified path
func deriveKey(masterKey *bip32.Key, path string) (*ecdsa.PrivateKey, error) {
	// Parse the derivation path
	derivationPath, err := accounts.ParseDerivationPath(path)
	if err != nil {
		return nil, err
	}

	// Derive the key
	key := masterKey
	for _, n := range derivationPath {
		key, err = key.NewChildKey(n)
		if err != nil {
			return nil, err
		}
	}

	// Convert to ecdsa.PrivateKey
	privateKey, err := crypto.ToECDSA(key.Key)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

// getAddressFromKey generates an Ethereum address from the private key
func getAddressFromKey(key *ecdsa.PrivateKey) common.Address {
	publicKey := key.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal().Msg("error casting public key to ECDSA")
	}

	return crypto.PubkeyToAddress(*publicKeyECDSA)
}
