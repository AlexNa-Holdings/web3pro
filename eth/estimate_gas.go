package eth

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum"
	"github.com/rs/zerolog/log"
)

func estimateGas(msg *bus.Message) (string, error) {
	req, ok := msg.Data.(*bus.B_EthEstimateGas)
	if !ok {
		return "", fmt.Errorf("invalid tx: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return "", errors.New("no wallet")
	}

	b := w.GetBlockchain(req.Blockchain)
	if b == nil {
		return "", fmt.Errorf("blockchain not found: %v", req.Blockchain)
	}

	from := w.GetAddress(req.From.String())
	if from == nil {
		return "", fmt.Errorf("address from not found: %v", req.From)
	}

	c, ok := cons[b.ChainId]
	if !ok {
		log.Error().Msgf("EstimateGas: Client not found for chainId: %d", b.ChainId)
		return "", fmt.Errorf("client not found for chainId: %d", b.ChainId)
	}

	// estimate gas
	call_msg := ethereum.CallMsg{
		From:  from.Address,
		To:    &req.To,
		Value: req.Amount,
		Data:  req.Data,
	}
	gas, err := c.EstimateGas(context.Background(), call_msg)
	if err != nil {
		log.Error().Msgf("EstimateGas: Cannot estimate gas. Error:(%v)", err)
		return "", err
	}

	hex_gas := fmt.Sprintf("0x%x", gas)

	return hex_gas, nil
}
