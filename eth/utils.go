package eth

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
)

func OpenClient(b *cmn.Blockchain) error {

	log.Trace().Msgf("OpenClient: Opening client for (%s)", b.Name)

	if b.Client != nil {
		return nil
	}

	client, err := ethclient.Dial(b.Url)
	if err != nil {
		return fmt.Errorf("OpenClient: Cannot dial to (%s). Error:(%v)", b.Url, err)
	}

	log.Trace().Msgf("OpenClient: Client opened to (%s)", b.Url)

	b.Client = client
	return nil
}

func BalanceOf(b *cmn.Blockchain, t *cmn.Token, address common.Address) (*big.Int, error) {
	if b.Name != t.Blockchain {
		return nil, fmt.Errorf("BalanceOf: Token (%s) is not from blockchain (%s)", t.Name, b.Name)
	}

	if t.Native {
		return GetBalance(b, address)
	} else {
		return GetERC20Balance(b, t, address)
	}
}

func SendTx(b *cmn.Blockchain, s *cmn.Signer, tx *types.Transaction, from *cmn.Address) error {
	// TODO: Implement this
	// d, err := s.GetDriver()
	// if err != nil {
	// 	log.Error().Msgf("Transfer: Cannot get driver. Error:(%v)", err)
	// 	return err
	// }

	// signedTx, err := d.SignTx(b, s, tx, from)
	// if err != nil {
	// 	log.Error().Msgf("Transfer: Cannot sign transaction. Error:(%v)", err)
	// 	return err
	// }

	// // Send the transaction
	// err = b.Client.SendTransaction(context.Background(), signedTx)
	// if err != nil {
	// 	log.Error().Msgf("Transfer: Cannot send transaction. Error:(%v)", err)
	// 	return err
	// }

	// cmn.Notifyf("Transaction sent: %s", signedTx.Hash().Hex())
	return nil
}
