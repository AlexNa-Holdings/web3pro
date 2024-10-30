package trezor

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/hw/trezor/trezorproto"
	"github.com/ava-labs/coreth/accounts"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
)

func signTx(msg *bus.Message) (*types.Transaction, error) {
	w := cmn.CurrentWallet
	if w == nil {
		return nil, errors.New("no wallet")
	}

	m, _ := msg.Data.(*bus.B_SignerSignTx)

	t := provide_device(msg, m.Name)
	if t == nil {
		return nil, fmt.Errorf("Trezor not found: %s", m.Name)
	}

	dp, err := accounts.ParseDerivationPath(m.Path)
	if err != nil {
		log.Error().Err(err).Msgf("SignTypedData: Error parsing path: %s", m.Path)
		return nil, err
	}

	b := w.GetBlockchainByName(m.Chain)
	if b == nil {
		return nil, fmt.Errorf("sign_tx: blockchain not found: %v", m.Chain)
	}

	ch_id := uint32(b.ChainId)
	to := m.Tx.To().Hex()

	request := &trezorproto.EthereumSignTxEIP1559{
		AddressN:       dp,
		Nonce:          new(big.Int).SetUint64(m.Tx.Nonce()).Bytes(),
		MaxGasFee:      m.Tx.GasFeeCap().Bytes(),
		MaxPriorityFee: m.Tx.GasTipCap().Bytes(),
		ChainId:        &ch_id,
		GasLimit:       new(big.Int).SetUint64(m.Tx.Gas()).Bytes(),
		Value:          m.Tx.Value().Bytes(),
		To:             &to,
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
	request.ChainId = &ch_id // EIP-155 transaction, set chain ID explicitly (only 32 bit is supported!?)

	response := new(trezorproto.EthereumTxRequest)

	save_mode := t.Pane.Mode
	save_template := t.Pane.GetTemplate()
	defer func() {
		t.Pane.SetTemplate(save_template)
		t.Pane.SetMode(save_mode)
	}()

	t.Pane.SetTemplate("<w><c>\nPlease <blink>sign</blink> transaction on your Trezor device\n")
	t.Pane.SetMode("template")

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

	// v_bytes := make([]byte, 2)
	// binary.BigEndian.PutUint16(v_bytes, uint16(ch_id*2+35))
	// //	binary.BigEndian.PutUint16(v_bytes, uint16(response.GetSignatureV()))

	// signature := append(append(response.GetSignatureR(), response.GetSignatureS()...), v_bytes...)
	signature := append(append(response.GetSignatureR(), response.GetSignatureS()...), byte(response.GetSignatureV()))

	//	signedTx, err := m.Tx.WithSignature(types.NewCancunSigner(big.NewInt(m.Tx.ChainId().Int64())), signature)
	signedTx, err := m.Tx.WithSignature(types.NewCancunSigner(big.NewInt(int64(ch_id))), signature)
	if err != nil {
		log.Error().Err(err).Msg("signTx: Failed to sign transaction")
		return nil, err
	}

	return signedTx, nil
}
