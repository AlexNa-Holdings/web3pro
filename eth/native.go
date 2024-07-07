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

	err := OpenClient(b)
	if err != nil {
		log.Error().Msgf("GetBalance: Failed to open client: %v", err)
		return nil, err
	}

	balance, err := b.Client.BalanceAt(context.Background(), address, nil)
	if err != nil {
		log.Error().Msgf("GetBalance: Cannot get balance. Error:(%v)", err)
		return nil, err
	}

	return balance, nil
}

func Transfer(b *cmn.Blockchain, s *cmn.Signer, from *cmn.Address, to common.Address, amount *big.Int) error {

	log.Trace().Msgf("Transfer: Token:(%s) Blockchain:(%s) From:(%s) To:(%s) Amount:(%s)", b.Currency, b.Name, from.Address.String(), to.String(), amount.String())

	if from.Signer != s.Name {
		log.Error().Msgf("Transfer: Signer mismatch. Token:(%s) Blockchain:(%s)", from.Signer, s.Name)
		return errors.New("signer mismatch")
	}

	err := OpenClient(b)
	if err != nil {
		log.Error().Msgf("Transfer: Failed to open client: %v", err)
		return err
	}

	nonce, err := b.Client.PendingNonceAt(context.Background(), from.Address)
	if err != nil {
		log.Error().Msgf("Transfer: Cannot get nonce. Error:(%v)", err)
		return err
	}

	// Suggest gas price
	gasPrice, err := b.Client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Error().Msgf("Transfer: Cannot suggest gas price. Error:(%v)", err)
		return err
	}

	tx := types.NewTransaction(nonce, to, amount, uint64(21000), gasPrice, nil)

	d, err := s.GetDriver()
	if err != nil {
		log.Error().Msgf("Transfer: Cannot get driver. Error:(%v)", err)
		return err
	}

	signedTx, err := d.SignTx(b, s, tx, from)
	if err != nil {
		log.Error().Msgf("Transfer: Cannot sign transaction. Error:(%v)", err)
		return err
	}

	// Send the transaction
	err = b.Client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Error().Msgf("Transfer: Cannot send transaction. Error:(%v)", err)
		return err
	}

	cmn.Notifyf("Transaction sent: %s", signedTx.Hash().Hex())

	return nil
}
