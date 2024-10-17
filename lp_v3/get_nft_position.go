package lp_v3

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

func get_nft_position(msg *bus.Message) (*bus.B_LP_V3_GetNftPosition_Response, error) {
	req, ok := msg.Data.(*bus.B_LP_V3_GetNftPosition)
	if !ok {
		return nil, fmt.Errorf("get_position: invalid data: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("get_position: no wallet")
	}

	data, err := V3_MANAGER.Pack("positions", req.NFT_Token)
	if err != nil {
		log.Error().Err(err).Msg("V3_ABI.Pack positions")
		return nil, err
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: req.ChainId,
		To:      req.Provider,
		From:    req.From,
		Data:    data,
	})

	if resp.Error != nil {
		log.Error().Err(resp.Error).Msg("eth call")
		return nil, resp.Error
	}

	var r_data bus.B_LP_V3_GetNftPosition_Response

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		log.Error().Err(err).Msg("hexutil.Decode")
		return nil, err
	}

	tick_l := new(big.Int)
	tick_u := new(big.Int)

	err = V3_MANAGER.UnpackIntoInterface(
		&[]interface{}{
			&r_data.Nonce,
			&r_data.Operator,
			&r_data.Token0,
			&r_data.Token1,
			&r_data.Fee,
			&tick_l,
			&tick_u,
			&r_data.Liquidity,
			&r_data.FeeGrowthInside0LastX128,
			&r_data.FeeGrowthInside1LastX128,
			&r_data.TokensOwed0,
			&r_data.TokensOwed1,
		}, "positions", output)

	if err != nil {
		log.Error().Err(err).Msg("positionManagerABI.UnpackIntoInterface")
		return nil, err
	}

	r_data.TickLower = tick_l.Int64()
	r_data.TickUpper = tick_u.Int64()

	return &r_data, nil
}
