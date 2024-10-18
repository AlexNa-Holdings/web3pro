package eth

import (
	"context"
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
)

func BalanceOf(b *cmn.Blockchain, t *cmn.Token, address common.Address) (*big.Int, error) {
	if b.ChainId != t.ChainId {
		return nil, fmt.Errorf("BalanceOf: Token (%s) is not from blockchain (%d)", t.Name, b.ChainId)
	}

	if t.Native {
		return GetBalance(b, address)
	} else {
		return GetERC20Balance(b, t, address)
	}
}

func SendSignedTx(signedTx *types.Transaction) (string, error) {

	c, ok := cons[int(signedTx.ChainId().Int64())]
	if !ok {
		log.Error().Msgf("SendSignedTx: Client not found for chainId: %v", signedTx.ChainId())
		return "", fmt.Errorf("client not found for chainId: %v", signedTx.ChainId())
	}

	// Send the transaction
	err := c.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Error().Err(err).Msgf("SendSignedTx: Cannot send transaction")
		return "", err
	}

	bus.Send("ui", "notify", fmt.Sprintf("Transaction sent: %s", signedTx.Hash().Hex()))

	return signedTx.Hash().Hex(), nil
}
