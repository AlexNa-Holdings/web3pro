package lp_v3

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

func get_fee_growth(msg *bus.Message) (*bus.B_LP_V3_GetFeeGrowth_Response, error) {
	req, ok := msg.Data.(*bus.B_LP_V3_GetFeeGrowth)
	if !ok {
		return nil, fmt.Errorf("get_fee_growth: invalid data: %v", msg.Data)
	}

	return _get_fee_growth(req.ChainId, req.Pool)
}

func _get_fee_growth(chain_id int, pool common.Address) (*bus.B_LP_V3_GetFeeGrowth_Response, error) {

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("get_fee_growth: no wallet")
	}

	// request feeGrowthGlobal0X128
	data, err := V3_POOL_UNISWAP.Pack("feeGrowthGlobal0X128")
	if err != nil {
		log.Error().Err(err).Msg("V3_ABI.Pack positions")
		return nil, err
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: chain_id,
		To:      pool,
		Data:    data,
	})

	if resp.Error != nil {
		log.Error().Err(resp.Error).Msg("eth call")
		return nil, resp.Error
	}

	output0, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		log.Error().Err(err).Msg("hexutil.Decode")
		return nil, err
	}

	// request feeGrowthGlobal1X128
	data, err = V3_POOL_UNISWAP.Pack("feeGrowthGlobal1X128")
	if err != nil {
		log.Error().Err(err).Msg("V3_ABI.Pack positions")
		return nil, err
	}

	resp = bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: chain_id,
		To:      pool,
		Data:    data,
	})

	if resp.Error != nil {
		log.Error().Err(resp.Error).Msg("eth call")
		return nil, resp.Error
	}

	output1, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		log.Error().Err(err).Msg("hexutil.Decode")
		return nil, err
	}

	return unpackFeeGrowth(output0, output1)
}

func unpackFeeGrowth(output0, outout1 []byte) (*bus.B_LP_V3_GetFeeGrowth_Response, error) {

	values, err := V3_POOL_UNISWAP.Unpack("feeGrowthGlobal0X128", output0)
	if err != nil {
		log.Error().Err(err).Msg("V3_POOL_ABI.Unpack slot0")
		return nil, err
	}

	fee0 := values[0].(*big.Int)

	values, err = V3_POOL_UNISWAP.Unpack("feeGrowthGlobal1X128", outout1)
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
