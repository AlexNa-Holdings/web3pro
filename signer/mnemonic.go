package signer

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

type MnemonicDriver struct {
	MasterKey *bip32.Key
}

func NewMnemonicDriver() MnemonicDriver {
	return MnemonicDriver{}
}

func (d MnemonicDriver) GetName(path string) (string, error) {
	return "", nil
}

func (d MnemonicDriver) GetMasterKey(s *cmn.Signer) (*bip32.Key, error) {
	entropy, err := hex.DecodeString(s.SN)
	if err != nil {
		log.Error().Msgf("GetMasterKey: Error decoding entropy: %v", err)
		return nil, err
	}

	mnemonics, err := bip39.NewMnemonic(entropy)
	if err != nil {
		log.Error().Msgf("GetMasterKey: Error creating mnemonics: %v", err)
		return nil, err
	}

	seed := bip39.NewSeed(mnemonics, "")
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		log.Error().Msgf("GetMasterKey: Error creating master key: %v", err)
		return nil, err
	}

	return masterKey, nil
}

func (d MnemonicDriver) GetAddresses(s *cmn.Signer, path_format string, start_from int, count int) ([]cmn.Address, error) {
	addresses := []cmn.Address{}

	if !strings.Contains(path_format, "%d") {
		return addresses, errors.New("path_format must contain %d")
	}

	masterKey, err := d.GetMasterKey(s)
	if err != nil {
		log.Error().Msgf("Error getting master key: %v", err)
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

func (d MnemonicDriver) SignTx(b *cmn.Blockchain, s *cmn.Signer, tx *types.Transaction, a *cmn.Address) (*types.Transaction, error) {
	masterKey, err := d.GetMasterKey(s)
	if err != nil {
		log.Error().Msgf("SignTx: Error getting master key: %v", err)
		return nil, err
	}

	// Get the private key
	privateKey, err := cmn.DeriveKey(masterKey, a.Path)
	if err != nil {
		log.Error().Msgf("SignTx: Failed to derive key: %v", err)
		return nil, err
	}

	// Sign the transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(int64(b.ChainId))), privateKey)
	if err != nil {
		log.Error().Msgf("SignTx: Failed to sign transaction: %v", err)
	}

	return signedTx, nil

}
