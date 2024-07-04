package signer_driver

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/rs/zerolog/log"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

type MnemonicDriver struct {
}

func NewMnemonicDriver() MnemonicDriver {
	return MnemonicDriver{}
}

func (d MnemonicDriver) GetName(path string) (string, error) {
	return "", nil
}

func (d MnemonicDriver) GetAddresses(s *cmn.Signer, path_format string, start_from int, count int) ([]cmn.Address, error) {
	addresses := []cmn.Address{}

	if !strings.Contains(path_format, "%d") {
		return addresses, errors.New("path_format must contain %d")
	}

	entropy, err := hex.DecodeString(s.SN)
	if err != nil {
		return addresses, err
	}

	mnemonics, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return addresses, err
	}

	seed := bip39.NewSeed(mnemonics, "")
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return addresses, err
	}

	for i := 0; i < count; i++ {
		p := fmt.Sprintf(path_format, start_from+i)

		key, err := cmn.DeriveKey(masterKey, p)
		if err != nil {
			log.Error().Msgf("Error deriving key: %v", err)
			return addresses, err
		}

		// Get the Ethereum address
		a := cmn.GetAddressFromKey(key)
		addresses = append(addresses, cmn.Address{
			Address: a,
			Signer:  s.Name,
			Path:    p,
		})
	}

	return addresses, nil
}

func (d MnemonicDriver) IsConnected(signer *cmn.Signer) bool {
	return true
}

func (d MnemonicDriver) PrintDetails(path string) string {
	return ""
}
