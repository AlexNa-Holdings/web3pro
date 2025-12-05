package eth

import (
	"context"
	"errors"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
)

func GetBalance(b *cmn.Blockchain, address common.Address) (*big.Int, error) {

	// Apply rate limiting for direct RPC call
	acquireRateLimit(b.ChainId)

	client, err := getEthClient(b)
	if err != nil {
		log.Error().Msgf("GetBalance: Failed to open client: %v", err)
		return nil, err
	}

	balance, err := client.BalanceAt(context.Background(), address, nil)
	handleRPCResult(b.ChainId, err)
	if err != nil {
		log.Error().Msgf("GetBalance: Cannot get balance. Error:(%v)", err)
		return nil, err
	}

	return balance, nil
}

func BuildTxTransfer(b *cmn.Blockchain, s *cmn.Signer, from *cmn.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	if from.Signer != s.Name {
		log.Error().Msgf("BuildTxTransfer: Signer mismatch. Token:(%s) Blockchain:(%s)", from.Signer, s.Name)
		return nil, errors.New("signer mismatch")
	}

	client, err := getEthClient(b)
	if err != nil {
		log.Error().Msgf("BuildTxTransfer: Failed to open client: %v", err)
		return nil, err
	}

	nonce, err := client.PendingNonceAt(context.Background(), from.Address)
	if err != nil {
		log.Error().Msgf("BuildTxTransfer: Cannot get nonce. Error:(%v)", err)
		return nil, err
	}

	gasLimit, err := client.EstimateGas(context.Background(), ethereum.CallMsg{
		From:  from.Address,
		To:    &to,
		Data:  nil,
		Value: amount,
		Gas:   0,
	})
	if err != nil {
		log.Error().Msgf("BuildTxERC20Transfer: Cannot estimate gas. Error:(%v)", err)
		return nil, err
	}

	priorityFee, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("Failed to suggest gas tip cap")
		return nil, err
	}

	// Get the latest block to determine the base fee
	block, err := client.BlockByNumber(context.Background(), nil) // Get the latest block
	if err != nil {
		log.Error().Err(err).Msg("Failed to get the latest block")
		return nil, err
	}

	// Base fee is included in the block header (introduced in EIP-1559)
	baseFee := block.BaseFee()
	// Calculate the MaxFeePerGas based on base fee and priority fee
	// For example, you might want to set MaxFeePerGas to be slightly higher than baseFee + priorityFee
	maxFeePerGas := new(big.Int).Add(baseFee, priorityFee)
	buffer := big.NewInt(2) // Set a buffer (optional) to ensure transaction gets processed
	maxFeePerGas = maxFeePerGas.Mul(maxFeePerGas, buffer)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(int64(b.ChainId)),
		Nonce:     nonce,
		To:        &to,
		Value:     amount,
		Gas:       gasLimit,
		GasFeeCap: maxFeePerGas,
		GasTipCap: priorityFee,
		Data:      nil,
	})

	return tx, nil

}

func Transfer(msg *bus.Message, b *cmn.Blockchain, s *cmn.Signer, from *cmn.Address, to common.Address, amount *big.Int) error {

	log.Trace().Msgf("Transfer: Token:(%s) Blockchain:(%s) From:(%s) To:(%s) Amount:(%s)", b.Currency, b.Name, from.Address.String(), to.String(), amount.String())

	tx, err := BuildTxTransfer(b, s, from, to, amount)
	if err != nil {
		log.Error().Msgf("Transfer: Cannot build transaction. Error:(%v)", err)
		return err
	}

	res := msg.Fetch("signer", "sign-tx", &bus.B_SignerSignTx{
		Type:      s.Type,
		Name:      s.Name,
		MasterKey: s.MasterKey,
		Chain:     b.Name,
		Tx:        tx,
		From:      from.Address,
		Path:      from.Path,
	})

	if res.Error != nil {
		log.Error().Err(res.Error).Msg("Transfer: Cannot sign tx")
		return res.Error
	}

	signedTx, ok := res.Data.(*types.Transaction)
	if !ok {
		log.Error().Msgf("Transfer: Cannot convert to transaction. Data:(%v)", res.Data)
		return errors.New("cannot convert to transaction")
	}

	hash, err := SendSignedTx(signedTx)
	if err != nil {
		log.Error().Err(err).Msg("Transfer: Cannot send tx")
		return err
	}

	bus.Send("ui", "notify", "Transaction sent: "+hash)

	return nil
}
