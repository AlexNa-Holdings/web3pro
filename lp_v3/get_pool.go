package lp_v3

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

func get_pool(msg *bus.Message) (common.Address, error) {
	req, ok := msg.Data.(*bus.B_LP_V3_GetPool)
	if !ok {
		return common.Address{}, fmt.Errorf("get_pool: invalid data: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return common.Address{}, fmt.Errorf("get_factory: no wallet")
	}

	data, err := V3_FACTORY.Pack("getPool", req.Token0, req.Token1, req.Fee)
	if err != nil {
		log.Error().Err(err).Msg("V3_FACTORY.Pack getPool")
		return common.Address{}, err
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: req.ChainId,
		To:      req.Factory,
		From:    w.CurrentAddress,
		Data:    data,
	})

	if resp.Error != nil {
		log.Error().Err(resp.Error).Msg("get_factory: eth call")
		return common.Address{}, resp.Error
	}

	pool := common.HexToAddress(resp.Data.(string))

	return pool, nil
}
