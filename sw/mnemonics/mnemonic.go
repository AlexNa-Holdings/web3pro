package mnemonics

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ava-labs/coreth/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/rs/zerolog/log"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

type Mnemonic struct {
	MasterKey *bip32.Key
}

func Loop() {
	ch := bus.Subscribe("signer")
	for msg := range ch {
		if msg.RespondTo != 0 {
			continue // ignore responses
		}
		go process(msg)
	}
}

func process(msg *bus.Message) {
	w := cmn.CurrentWallet
	if w == nil {
		msg.Respond(nil, errors.New("no wallet"))
		return
	}

	switch msg.Topic {
	case "signer":
		switch msg.Type {
		case "is-connected":
			m, ok := msg.Data.(*bus.B_SignerIsConnected)
			if !ok {
				log.Error().Msg("Loop: Invalid hw is-connected data")
				return
			}

			if m.Type == "mnemonics" {
				msg.Respond(&bus.B_SignerIsConnected_Response{Connected: true}, nil)
			}

		case "get-addresses":
			m, ok := msg.Data.(*bus.B_SignerGetAddresses)
			if !ok {
				log.Error().Msg("Loop: Invalid hw get-addresses data")
				return
			}

			if m.Type == "mnemonics" {
				mnemonics, err := NewFromSN(m.MasterKey)
				if err != nil {
					log.Error().Msgf("Error creating mnemonics: %v", err)
					msg.Respond(&bus.B_SignerGetAddresses_Response{}, err)
					return
				}

				a, p, err := mnemonics.GetAddresses(m.Path, m.StartFrom, m.Count)
				if err != nil {
					log.Error().Msgf("Error getting addresses: %v", err)
					msg.Respond(&bus.B_SignerGetAddresses_Response{}, err)
					return
				}
				msg.Respond(&bus.B_SignerGetAddresses_Response{
					Addresses: a,
					Paths:     p,
				}, nil)

			}
		case "sign-tx":
			m, ok := msg.Data.(*bus.B_SignerSignTx)
			if !ok {
				log.Error().Msg("Loop: Invalid hw sign-tx data")
				msg.Respond(nil, errors.New("invalid data"))
				return
			}

			if m.Type == "mnemonics" {
				b := w.GetBlockchain(m.Chain)
				if b == nil {
					log.Error().Msgf("Error getting blockchain: %v", m.Chain)
					msg.Respond(nil, fmt.Errorf("blockchain not found: %v", m.Chain))
					return
				}

				mnemonics, err := NewFromSN(m.MasterKey)
				if err != nil {
					log.Error().Msgf("Error creating mnemonics: %v", err)
					msg.Respond(nil, err)
					return
				}

				tx, err := mnemonics.SignTx(int64(b.ChainId), m.Tx, m.Path)
				if err != nil {
					log.Error().Msgf("Error signing transaction: %v", err)
					msg.Respond(nil, err)
					return
				}

				msg.Respond(tx, nil)
			}
		case "sign-typed-data-v4":
			m, ok := msg.Data.(*bus.B_SignerSignTypedData_v4)
			if !ok {
				log.Error().Msg("Loop: Invalid hw sign-typed-data-v4 data")
				msg.Respond(nil, errors.New("invalid data"))
				return
			}

			if m.Type == "mnemonics" {
				mnemonics, err := NewFromSN(m.MasterKey)
				if err != nil {
					log.Error().Msgf("Error creating mnemonics: %v", err)
					msg.Respond(nil, err)
					return
				}

				// Sign the typed data
				signature, err := mnemonics.SignTypedData(msg, m.TypedData, m.Path)
				if err != nil {
					log.Error().Msgf("Error signing typed data: %v", err)
					msg.Respond(nil, err)
					return
				}

				msg.Respond(signature, nil)
			}

		}
	}
}

func NewFromSN(SN string) (*Mnemonic, error) {
	entropy, err := hex.DecodeString(SN)
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

	return &Mnemonic{
		MasterKey: masterKey,
	}, nil
}

// deriveKey derives a key from the master key using the specified path
func DeriveKey(masterKey *bip32.Key, path string) (*ecdsa.PrivateKey, error) {
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
func GetAddressFromKey(key *ecdsa.PrivateKey) common.Address {
	publicKey := key.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal().Msg("error casting public key to ECDSA")
	}

	return crypto.PubkeyToAddress(*publicKeyECDSA)
}

func (d *Mnemonic) GetAddresses(path_format string, start_from int, count int) ([]common.Address, []string, error) {
	addresses := []common.Address{}
	paths := []string{}

	if !strings.Contains(path_format, "%d") {
		return addresses, paths, errors.New("path_format must contain %d")
	}

	for i := 0; i < count; i++ {
		p := fmt.Sprintf(path_format, start_from+i)

		key, err := DeriveKey(d.MasterKey, p)
		if err != nil {
			log.Error().Msgf("Error deriving key: %v", err)
			return addresses, paths, err
		}

		// Get the Ethereum address
		a := GetAddressFromKey(key)
		addresses = append(addresses, a)
		paths = append(paths, p)
	}

	return addresses, paths, nil
}

func (d Mnemonic) SignTx(chain_id int64, tx *types.Transaction, path string) (*types.Transaction, error) {
	// Get the private key
	privateKey, err := DeriveKey(d.MasterKey, path)
	if err != nil {
		log.Error().Msgf("SignTx: Failed to derive key: %v", err)
		return nil, err
	}

	// Sign the transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(chain_id)), privateKey)
	if err != nil {
		log.Error().Msgf("SignTx: Failed to sign transaction: %v", err)
	}

	return signedTx, nil

}

func (d Mnemonic) SignTypedData(msg *bus.Message, typedData apitypes.TypedData, path string) (string, error) {
	// Get the private key
	privateKey, err := DeriveKey(d.MasterKey, path)
	if err != nil {
		log.Error().Msgf("SignTypedData: Failed to derive key: %v", err)
		return "", err
	}

	data, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		log.Error().Msgf("SignTypedData: Failed to hash typed data: %v", err)
		return "", err
	}

	// Sign the hash
	signature, err := crypto.Sign(data, privateKey)
	if err != nil {
		log.Error().Msgf("SignTypedData: Failed to sign hash: %v", err)
		return "", err
	}

	OK := false

	msg.Fetch("ui", "hail", &bus.B_Hail{
		Title:    "Sign Typed Data",
		Template: cmn.ConfirmEIP712Template(typedData),
		OnOk: func(m *bus.Message) {
			OK = true
			bus.Send("ui", "remove-hail", m)
		},
		OnOverHotspot: func(m *bus.Message, v *gocui.View, hs *gocui.Hotspot) {
			cmn.StandardOnOverHotspot(v, hs)
		},
		OnClickHotspot: func(m *bus.Message, v *gocui.View, hs *gocui.Hotspot) {
			cmn.StandardOnClickHotspot(v, hs)
		},
	})

	if OK {
		ss := fmt.Sprintf("0x%x", signature)
		log.Info().Msgf("Signature: %s", ss)
		return ss, nil
	} else {
		return "", errors.New("cancelled")
	}
}
