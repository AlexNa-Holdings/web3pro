package signer

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"

	"github.com/AlexNa-Holdings/web3pro/address"
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
	}, nil
}

func (d MnemonicDriver) GetAddresses(start_from int, count int) ([]address.Address, error) {
	addresses := []address.Address{}

	mnemonics, err := bip39.NewMnemonic(d.Entropy)
	if err != nil {
		return addresses, err
	}

	seed := bip39.NewSeed(mnemonics, "")
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return addresses, err
	}

	for i := 0; i < count; i++ { // Generate first 5 addresses
		// Derive the key using the path m/44'/60'/0'/0/i
		path := fmt.Sprintf("m/44'/60'/0'/0/%d", start_from+i)
		key, err := deriveKey(masterKey, path)
		if err != nil {
			log.Fatal(err)
		}

		// Get the Ethereum address
		a := getAddressFromKey(key)
		addresses = append(addresses, address.Address{
			Address: a,
			Signer:  d.Name,
			Path:    path,
		})
	}

	return addresses, nil
}

func (d MnemonicDriver) IsConnected() bool {
	return true
}
