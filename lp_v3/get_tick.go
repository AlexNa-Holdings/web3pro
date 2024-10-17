package lp_v3

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

func get_tick(msg *bus.Message) (*bus.B_LP_V3_GetTick_Response, error) {
	req, ok := msg.Data.(*bus.B_LP_V3_GetTick)
	if !ok {
		return nil, fmt.Errorf("get_slot0: invalid data: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("get_slot0: no wallet")
	}

	data, err := V3_POOL_UNISWAP.Pack("ticks", big.NewInt(int64(req.Tick)))
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

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		log.Error().Err(err).Msg("hexutil.Decode")
		return nil, err
	}

	// Unpack the data into an interface slice
	values, err := V3_POOL_UNISWAP.Unpack("ticks", output)
	if err != nil {
		log.Error().Msgf("Failed to unpack tick data: %v", err)
		return nil, err
	}

	return &bus.B_LP_V3_GetTick_Response{
		LiquidityGross:                 values[0].(*big.Int),
		LiquidityNet:                   values[1].(*big.Int),
		FeeGrowthOutside0X128:          values[2].(*big.Int),
		FeeGrowthOutside1X128:          values[3].(*big.Int),
		TickCumulativeOutside:          values[4].(*big.Int),
		SecondsPerLiquidityOutsideX128: values[5].(*big.Int),
		SecondsOutside:                 uint32(values[6].(uint32)),
		Initialized:                    values[7].(bool),
	}, nil
}
