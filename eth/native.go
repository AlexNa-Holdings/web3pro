package eth

import (
	"context"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
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
