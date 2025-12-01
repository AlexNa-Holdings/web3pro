package lp_v2

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

func getPair(msg *bus.Message) (common.Address, error) {
	req, ok := msg.Data.(*bus.B_LP_V2_GetPair)
	if !ok {
		return common.Address{}, fmt.Errorf("get_pair: invalid data: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return common.Address{}, fmt.Errorf("get_pair: no wallet")
	}

	data, err := V2_FACTORY.Pack("getPair", req.Token0, req.Token1)
	if err != nil {
		log.Error().Err(err).Msg("V2_FACTORY.Pack getPair")
		return common.Address{}, err
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: req.ChainId,
		To:      req.Factory,
		Data:    data,
	})

	if resp.Error != nil {
		log.Error().Err(resp.Error).Msg("eth call getPair")
		return common.Address{}, resp.Error
	}

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		log.Error().Err(err).Msg("hexutil.Decode getPair")
		return common.Address{}, err
	}

	var pair common.Address
	err = V2_FACTORY.UnpackIntoInterface(&pair, "getPair", output)
	if err != nil {
		log.Error().Err(err).Msg("V2_FACTORY.UnpackIntoInterface getPair")
		return common.Address{}, err
	}

	return pair, nil
}
