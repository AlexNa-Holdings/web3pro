package lp_v3

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

func get_slot0(msg *bus.Message) (*bus.B_LP_V3_GetSlot0_Response, error) {
	req, ok := msg.Data.(*bus.B_LP_V3_GetSlot0)
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

	if err != nil {
		log.Error().Err(err).Msg("V3_POOL_ABI.UnpackIntoInterface slot0")
		return nil, err
	}

	return &bus.B_LP_V3_GetSlot0_Response{
		SqrtPriceX96: values[0].(*big.Int),
		Tick:         values[1].(*big.Int),
	}, nil
}
