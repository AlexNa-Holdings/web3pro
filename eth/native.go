package eth

import (
	"context"
	"errors"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
)

func GetBalance(b *cmn.Blockchain, address common.Address) (*big.Int, error) {

	client, err := getEthClient(b)
	if err != nil {
		log.Error().Msgf("GetBalance: Failed to open client: %v", err)
		return nil, err
	}

	balance, err := client.BalanceAt(context.Background(), address, nil)
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

	// Suggest gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Error().Msgf("BuildTxTransfer: Cannot suggest gas price. Error:(%v)", err)
		return nil, err
	}

	tx := types.NewTransaction(nonce, to, amount, uint64(21000), gasPrice, nil)
	return tx, nil

}

func Transfer(b *cmn.Blockchain, s *cmn.Signer, from *cmn.Address, to common.Address, amount *big.Int) error {

	log.Trace().Msgf("Transfer: Token:(%s) Blockchain:(%s) From:(%s) To:(%s) Amount:(%s)", b.Currency, b.Name, from.Address.String(), to.String(), amount.String())

	tx, err := BuildTxTransfer(b, s, from, to, amount)
	if err != nil {
		log.Error().Msgf("Transfer: Cannot build transaction. Error:(%v)", err)
		return err
	}

	err = SendTx(b, s, tx, from)
	if err != nil {
		log.Error().Msgf("Transfer: Cannot send transaction. Error:(%v)", err)
		return err
	}

	return nil
}
