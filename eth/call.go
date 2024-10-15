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

func call(msg *bus.Message) (bool, string, error) {
	req, ok := msg.Data.(*bus.B_EthCall)
	if !ok {
		return true, "", fmt.Errorf("invalid tx: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return true, "", errors.New("no wallet")
	}

	b := w.GetBlockchainById(req.ChainId)
	if b == nil {
		return true, "", fmt.Errorf("blockchain not found: %v", req.ChainId)
	}

	c, ok := cons[b.ChainId]
	if !ok {
		log.Error().Msgf("SendSignedTx: Client not found for chainId: %d", b.ChainId)
		return true, "", fmt.Errorf("client not found for chainId: %d", b.ChainId)
	}

	// Multicall agregation is OFF
	// if b.Multicall.Cmp(common.Address{}) != 0 {
	// 	c.Multicall.Add(msg)
	// 	return false, "", nil
	// }

	call_msg := ethereum.CallMsg{
		To:    &req.To,
		From:  req.From,
		Value: req.Amount,
		Data:  req.Data,
	}

	output, err := c.CallContract(context.Background(), call_msg, nil)
	if err != nil {
		log.Error().Msgf("call: Cannot call contract. Error:(%v)", err)
		return true, "", err
	}

	hex_data := fmt.Sprintf("0x%x", output)

	return true, hex_data, nil
}
