package ledger

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/rs/zerolog/log"
)

func signTx(msg *bus.Message) (*types.Transaction, error) {
	m, _ := msg.Data.(*bus.B_SignerSignTx)

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("SignTx: no wallet")
	}

	b := w.GetBlockchainByName(m.Chain)
	if b == nil {
		return nil, fmt.Errorf("SignTx: blockchain not found: %v", m.Chain)
	}

	ledger := provide_device(msg, m.Name)
	if ledger == nil {
		return nil, fmt.Errorf("SignTx: no device found with name %s", m.Name)
	}

	err := provide_eth_app(msg, ledger.USB_ID, "Ethereum")
	if err != nil {
		return nil, err
	}

	var payload []byte
	dp, err := accounts.ParseDerivationPath(m.Path)
	if err != nil {
		log.Error().Err(err).Msgf("SignTx: Error parsing path: %s", m.Path)
		return nil, err
	}
	payload = append(payload, serializePath(dp)...)

	unsignedTxBytes, err := serializeTxForLedger(m.Tx, big.NewInt(int64(b.ChainId)))
	if err != nil {
		log.Error().Err(err).Msg("SignTx: Failed to serialize transaction")

		return nil, err
	}
	payload = append(payload, unsignedTxBytes...)

	save_mode := ledger.Pane.Mode
	save_template := ledger.Pane.GetTemplate()
	defer func() {
		ledger.Pane.SetTemplate(save_template)
		ledger.Pane.SetMode(save_mode)
	}()

	ledger.Pane.SetTemplate("<w><c>\nPlease <blink>sign</blink> the transaction on your device\n")
	ledger.Pane.SetMode("template")

	reply, err := call(msg, ledger.USB_ID, &SIGN_TX_APDU, payload)
	if err != nil {
		log.Error().Err(err).Msgf("SignTx: Error signing transaction: %s", ledger.USB_ID)
		return nil, err
	}

	var sig []byte
	sig = append(sig, reply[1:]...) // R + S
	sig = append(sig, reply[0])     // V

	// sig[64] -= byte(b.ChainID*2 + 35)

	signedTx, err := m.Tx.WithSignature(types.NewCancunSigner(big.NewInt(int64(b.ChainId))), sig)
	if err != nil {
		log.Error().Err(err).Msg("signTx: Failed to sign transaction")
		return nil, err
	}

	return signedTx, nil
}

func serializeTxForLedger(tx *types.Transaction, chainID *big.Int) ([]byte, error) {
	var txType byte
	var txData []byte
	var err error

	switch tx.Type() {
	case types.LegacyTxType:
		// Legacy transaction
		txType = 0x00 // No type prefix
		txData, err = rlp.EncodeToBytes([]interface{}{
			tx.Nonce(),
			tx.GasPrice(),
			tx.Gas(),
			tx.To(),
			tx.Value(),
			tx.Data(),
		})
	case types.AccessListTxType:
		// EIP-2930 transaction
		txType = types.AccessListTxType
		txData, err = rlp.EncodeToBytes([]interface{}{
			chainID,
			tx.Nonce(),
			tx.GasPrice(),
			tx.Gas(),
			tx.To(),
			tx.Value(),
			tx.Data(),
			tx.AccessList(),
		})
	case types.DynamicFeeTxType:
		// EIP-1559 transaction
		txType = types.DynamicFeeTxType
		txData, err = rlp.EncodeToBytes([]interface{}{
			chainID,
			tx.Nonce(),
			tx.GasTipCap(),
			tx.GasFeeCap(),
			tx.Gas(),
			tx.To(),
			tx.Value(),
			tx.Data(),
			tx.AccessList(),
		})
	default:
		return nil, fmt.Errorf("unsupported transaction type: %d", tx.Type())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to RLP encode transaction: %v", err)
	}

	// For typed transactions, prepend the transaction type
	if txType != 0x00 {
		txData = append([]byte{txType}, txData...)
	}

	return txData, nil
}
