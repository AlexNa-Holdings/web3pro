package trezor

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/hw/trezor/trezorproto"
	"github.com/ava-labs/coreth/accounts"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
)

func signTx(msg *bus.Message) (*types.Transaction, error) {
	m, _ := msg.Data.(*bus.B_SignerSignTx)

	t := provide_device(m.Name)
	if t == nil {
		return nil, fmt.Errorf("Trezor not found: %s", m.Name)
	}

	dp, err := accounts.ParseDerivationPath(m.Path)
	if err != nil {
		log.Error().Err(err).Msgf("SignTypedData: Error parsing path: %s", m.Path)
		return nil, err
	}

	// request := &trezorproto.EthereumSignTx{
	// 	AddressN: dp,
	// 	Nonce:    new(big.Int).SetUint64(m.Tx.Nonce()).Bytes(),
	// 	GasPrice: m.Tx.GasPrice().Bytes(),
	// 	GasLimit: new(big.Int).SetUint64(m.Tx.Gas()).Bytes(),
	// 	Value:    m.Tx.Value().Bytes(),
	// }

	ch_id := uint32(m.Tx.ChainId().Uint64())

	request := &trezorproto.EthereumSignTxEIP1559{
		AddressN:       dp,
		Nonce:          new(big.Int).SetUint64(m.Tx.Nonce()).Bytes(),
		MaxGasFee:      m.Tx.GasFeeCap().Bytes(),
		MaxPriorityFee: m.Tx.GasTipCap().Bytes(),
		ChainId:        &ch_id,
		GasLimit:       new(big.Int).SetUint64(m.Tx.Gas()).Bytes(),
		Value:          m.Tx.Value().Bytes(),
	}

	data := m.Tx.Data()
	length := uint32(len(data))

	if to := m.Tx.To(); to != nil {
		// Non contract deploy, set recipient explicitly
		hex := to.Hex()
		request.To = &hex
	}

	if length > 1024 { // Send the data chunked if that was requested
		request.DataInitialChunk, data = data[:1024], data[1024:]
	} else {
		request.DataInitialChunk, data = data, nil
	}

	request.DataLength = &length

	if m.Tx.ChainId() != nil { // EIP-155 transaction, set chain ID explicitly (only 32 bit is supported!?)
		id := uint32(m.Tx.ChainId().Int64())
		request.ChainId = &id
	}

	response := new(trezorproto.EthereumTxRequest)

	if err := t.Call(msg, request, response); err != nil {
		log.Error().Err(err).Msgf("SignTx: Error signing typed data(1): %s", m.Path)
		return nil, err
	}

	for response.DataLength != nil && int(*response.DataLength) <= len(data) {
		chunk := data[:*response.DataLength]
		data = data[*response.DataLength:]

		if err := t.Call(msg, &trezorproto.EthereumTxAck{DataChunk: chunk}, response); err != nil {
			log.Error().Err(err).Msgf("SignTx: Error signing typed data(2): %s", m.Path)
			return nil, err
		}
	}

	// Extract the Ethereum signature and do a sanity validation
	if len(response.GetSignatureR()) == 0 || len(response.GetSignatureS()) == 0 {
		return nil, errors.New("trezor returned invalid signature")
	}
	signature := append(append(response.GetSignatureR(), response.GetSignatureS()...), byte(response.GetSignatureV()))

	log.Debug().Msgf("Signature: 0x%x", signature)
	log.Debug().Msgf("Len: %d", len(signature))
	log.Debug().Msgf("ChainID: %d", m.Tx.ChainId().Int64())

	signedTx, err := m.Tx.WithSignature(types.NewCancunSigner(big.NewInt(m.Tx.ChainId().Int64())), signature)
	if err != nil {
		log.Error().Err(err).Msg("signTx: Failed to sign transaction")
		return nil, err
	}

	return signedTx, nil
}
