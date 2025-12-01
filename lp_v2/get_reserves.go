package lp_v2

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

func getReserves(msg *bus.Message) (*bus.B_LP_V2_GetReserves_Response, error) {
	req, ok := msg.Data.(*bus.B_LP_V2_GetReserves)
	if !ok {
		return nil, fmt.Errorf("get_reserves: invalid data: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("get_reserves: no wallet")
	}

	data, err := V2_PAIR.Pack("getReserves")
	if err != nil {
		log.Error().Err(err).Msg("V2_PAIR.Pack getReserves")
		return nil, err
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: req.ChainId,
		To:      req.Pair,
		Data:    data,
	})

	if resp.Error != nil {
		log.Error().Err(resp.Error).Msg("eth call getReserves")
		return nil, resp.Error
	}

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		log.Error().Err(err).Msg("hexutil.Decode getReserves")
		return nil, err
	}

	// Unpack getReserves result: (uint112 reserve0, uint112 reserve1, uint32 blockTimestampLast)
	result, err := V2_PAIR.Unpack("getReserves", output)
	if err != nil {
		log.Error().Err(err).Msg("V2_PAIR.Unpack getReserves")
		return nil, err
	}

	if len(result) < 3 {
		return nil, fmt.Errorf("getReserves returned insufficient values")
	}

	reserve0 := result[0].(*big.Int)
	reserve1 := result[1].(*big.Int)
	// uint32 is unpacked directly as uint32, not *big.Int
	blockTimestampLast := result[2].(uint32)

	return &bus.B_LP_V2_GetReserves_Response{
		Reserve0:           reserve0,
		Reserve1:           reserve1,
		BlockTimestampLast: blockTimestampLast,
	}, nil
}
