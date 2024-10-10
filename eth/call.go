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

func call(msg *bus.Message) (string, error) {
	req, ok := msg.Data.(*bus.B_EthCall)
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

	c, ok := cons[b.ChainID]
	if !ok {
		log.Error().Msgf("SendSignedTx: Client not found for chainId: %d", b.ChainID)
		return "", fmt.Errorf("client not found for chainId: %d", b.ChainID)
	}

	call_msg := ethereum.CallMsg{
		To:    &req.To,
		From:  from.Address,
		Value: req.Amount,
		Data:  req.Data,
	}

	output, err := c.CallContract(context.Background(), call_msg, nil)
	if err != nil {
		log.Error().Msgf("call: Cannot call contract. Error:(%v)", err)
	}

	hex_data := fmt.Sprintf("0x%x", output)

	return hex_data, nil
}