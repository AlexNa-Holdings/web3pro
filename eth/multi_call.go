package eth

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

func multiCall(msg *bus.Message) ([][]byte, error) {
	req, ok := msg.Data.(*bus.B_EthMultiCall)
	if !ok {
		return nil, fmt.Errorf("invalid tx: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return nil, errors.New("no wallet")
	}

	b := w.GetBlockchainById(req.ChainId)
	if b == nil {
		return nil, fmt.Errorf("blockchain not found: %v", req.ChainId)
	}

	if b.Multicall == (common.Address{}) {
		return nil, fmt.Errorf("blockchain does not support multicall: %v", req.ChainId)
	}

	c, ok := cons[b.ChainId]
	if !ok {
		log.Error().Msgf("multiCall: Client not found for chainId: %d", b.ChainId)
		return nil, fmt.Errorf("client not found for chainId: %d", b.ChainId)
	}

	type Call struct {
		Target   common.Address
		CallData []byte
	}

	// Create a slice of Call type
	callArgs := []Call{}
	for _, c := range req.Calls {
		callArgs = append(callArgs, Call{
			Target:   c.To,
			CallData: c.Data,
		})
	}

	data, err := MULTICALL2_ABI.Pack("aggregate", callArgs)
	if err != nil {
		log.Error().Msgf("multicall: Cannot pack multicall data. Error:(%v)", err)
		return nil, err
	}

	call_msg := ethereum.CallMsg{
		To:    &b.Multicall,
		From:  req.From,
		Value: req.Amount,
		Data:  data,
	}

	output, err := c.CallContract(context.Background(), call_msg, nil)
	if err != nil {
		log.Error().Msgf("multicall: Cannot call contract. Error:(%v)", err)
		return nil, err
	}

	values, err := MULTICALL2_ABI.Unpack("aggregate", output)
	if err != nil {
		log.Error().Msgf("multicall: Cannot unpack aggregate data. Error: (%v)", err)
		return nil, err
	}

	returnData, ok := values[1].([][]byte)
	if !ok {
		log.Error().Msgf("multicall: Cannot convert return data to [][]byte")
		return nil, errors.New("cannot convert return data to [][]byte")
	}

	return returnData, nil
}
