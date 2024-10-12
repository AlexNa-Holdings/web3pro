package lp_v3

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

func get_factory(msg *bus.Message) (common.Address, error) {
	req, ok := msg.Data.(*bus.B_LP_V3_GetFactory)
	if !ok {
		return common.Address{}, fmt.Errorf("get_factory: invalid data: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return common.Address{}, fmt.Errorf("get_factory: no wallet")
	}

	data, err := V3_MANAGER.Pack("factory")
	if err != nil {
		log.Error().Err(err).Msg("V3_ABI.Pack positions")
		return common.Address{}, err
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: req.ChainId,
		To:      req.Provider,
		From:    w.CurrentAddress,
		Data:    data,
	})

	if resp.Error != nil {
		log.Error().Err(resp.Error).Msg("eth call")
		return common.Address{}, resp.Error
	}

	factory := common.HexToAddress(resp.Data.(string))

	return factory, nil
}
