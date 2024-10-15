package lp_v3

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

func get_fee_growth(msg *bus.Message) (*bus.B_LP_V3_GetFeeGrowth_Response, error) {
	req, ok := msg.Data.(*bus.B_LP_V3_GetFeeGrowth)
	if !ok {
		return nil, fmt.Errorf("get_fee_growth: invalid data: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("get_fee_growth: no wallet")
	}

	// request feeGrowthGlobal0X128
	data, err := V3_POOL.Pack("feeGrowthGlobal0X128")
	if err != nil {
		log.Error().Err(err).Msg("V3_ABI.Pack positions")
		return nil, err
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: req.ChainId,
		To:      req.Pool,
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

	values, err := V3_POOL.Unpack("feeGrowthGlobal0X128", output)
	if err != nil {
		log.Error().Err(err).Msg("V3_POOL_ABI.Unpack slot0")
		return nil, err
	}

	fee0 := values[0].(*big.Int)

	log.Debug().Msgf("---FEE_0: %v", fee0)

	// request feeGrowthGlobal1X128
	data, err = V3_POOL.Pack("feeGrowthGlobal1X128")
	if err != nil {
		log.Error().Err(err).Msg("V3_ABI.Pack positions")
		return nil, err
	}

	resp = bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: req.ChainId,
		To:      req.Pool,
		Data:    data,
	})

	if resp.Error != nil {
		log.Error().Err(resp.Error).Msg("eth call")
		return nil, resp.Error
	}

	output, err = hexutil.Decode(resp.Data.(string))
	if err != nil {
		log.Error().Err(err).Msg("hexutil.Decode")
		return nil, err
	}

	values, err = V3_POOL.Unpack("feeGrowthGlobal1X128", output)
	if err != nil {
		log.Error().Err(err).Msg("V3_POOL_ABI.Unpack slot0")
		return nil, err
	}

	fee1 := values[0].(*big.Int)

	return &bus.B_LP_V3_GetFeeGrowth_Response{
		FeeGrowthGlobal0X128: fee0,
		FeeGrowthGlobal1X128: fee1,
	}, nil
}
