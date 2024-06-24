package signer

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/address"
	"github.com/rs/zerolog/log"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

type MnemonicDriver struct {
	Entropy []byte
	*Signer
}

func NewMnemonicDriver(s *Signer) (MnemonicDriver, error) {

	if s.Type != "mnemonics" {
		return MnemonicDriver{}, errors.New("invalid signer type")
	}

	entropy, err := hex.DecodeString(s.SN)
	if err != nil {
		return MnemonicDriver{}, err
	}

	return MnemonicDriver{
		Entropy: entropy,
		Signer:  s,
	}, nil
}

func (d MnemonicDriver) GetAddresses(path string, start_from int, count int) ([]address.Address, error) {
	addresses := []address.Address{}

	if path == "" {
		path = "m/44'/60'/0'/0/"
	}

	mnemonics, err := bip39.NewMnemonic(d.Entropy)
	if err != nil {
		return addresses, err
	}

	seed := bip39.NewSeed(mnemonics, "")
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return addresses, err
	}

	log.Debug().Msgf("mnemonic: %s", mnemonics)

	for i := 0; i < count; i++ {
		p := fmt.Sprintf(path+"%d", start_from+i)

		log.Debug().Msgf("path: %s", p)

		key, err := deriveKey(masterKey, p)
		if err != nil {
			log.Error().Msgf("Error deriving key: %v", err)
			return addresses, err
		}

		// Get the Ethereum address
		a := getAddressFromKey(key)
		addresses = append(addresses, address.Address{
			Address: a,
			Signer:  d.Name,
			Path:    p,
		})
	}

	return addresses, nil
}

func (d MnemonicDriver) IsConnected() bool {
	return true
}
