package lp_v3

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

func get_price(msg *bus.Message) (*big.Int, error) {
	req, ok := msg.Data.(*bus.B_LP_V3_GetPrice)
	if !ok {
		return nil, fmt.Errorf("get_price: invalid data: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("get_price: no wallet")
	}

	data, err := V3_POOL.Pack("slot0")
	if err != nil {
		log.Error().Err(err).Msg("V3_ABI.Pack positions")
		return nil, err
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: req.ChainId,
		To:      req.Pool,
		From:    w.CurrentAddress,
		Data:    data,
	})

	if resp.Error != nil {
		log.Error().Err(resp.Error).Msg("eth call")
		return nil, resp.Error
	}

	log.Debug().Msgf("get_price: %v", req.Pool)

	// var (
	// 	sqrtPriceX96               *big.Int
	// 	tick                       *big.Int
	// 	observationIndex           uint16
	// 	observationCardinality     uint16
	// 	observationCardinalityNext uint16
	// 	feeProtocol                uint8
	// 	unlocked                   bool
	// )

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		log.Error().Err(err).Msg("hexutil.Decode")
		return nil, err
	}

	values, err := V3_POOL.Unpack("slot0", output)
	if err != nil {
		log.Error().Err(err).Msg("V3_POOL_ABI.Unpack slot0")
		return nil, err
	}

	// resultMap := map[string]interface{}{}
	// err = V3_POOL.UnpackIntoMap(resultMap, "slot0", output)
	// if err != nil {
	// 	log.Error().Err(err).Msg("V3_POOL_ABI.UnpackIntoMap slot0")
	// 	return nil, err
	// }

	// Use V3_POOL_ABI to unpack the result
	// err = V3_POOL.UnpackIntoInterface(
	// 	&[]interface{}{
	// 		&sqrtPriceX96,
	// 		&tick,
	// 		&observationIndex,
	// 		&observationCardinality,
	// 		&observationCardinalityNext,
	// 		&feeProtocol,
	// 		&unlocked,
	// 	}, "slot0", output)
	if err != nil {
		log.Error().Err(err).Msg("V3_POOL_ABI.UnpackIntoInterface slot0")
		return nil, err
	}

	return values[0].(*big.Int), nil
}
