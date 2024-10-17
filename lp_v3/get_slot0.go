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
		return nil, fmt.Errorf("get_slot0: invalid data: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("get_slot0: no wallet")
	}

	data, err := V3_POOL_UNISWAP.Pack("slot0")
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

	values, err := V3_POOL_UNISWAP.Unpack("slot0", output)
	if err != nil {
		values, err = V3_POOL_PANCAKE.Unpack("slot0", output)
		if err != nil {
			log.Error().Err(err).Msg("V3_POOL_ABI.Unpack slot0")
			return nil, err
		}
	}

	if err != nil {
		log.Error().Err(err).Msg("V3_POOL_ABI.UnpackIntoInterface slot0")
		return nil, err
	}

	if len(values) < 7 {
		return nil, fmt.Errorf("invalid slot0 values: %v", values)
	}

	log.Debug().Msgf("---SLOT0: %v", values)

	sqrtPriceX96, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid sqrtPriceX96: %v", values[0])
	}

	tick, ok := values[1].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid tick: %v", values[1])
	}

	feeProtocol0 := float32(0)
	feeProtocol1 := float32(0)
	feeProtocol8, ok := values[5].(uint8)
	if ok {
		if float32(feeProtocol8&0x0f) != 0 {
			feeProtocol0 = 100. / float32(feeProtocol8&0x0f)
		}

		if float32((feeProtocol8>>4)&0x0f) != 0 {
			feeProtocol1 = 100. / float32((feeProtocol8>>4)&0x0f)
		}

	} else {
		feeProtocol32, ok := values[5].(uint32)
		if ok {

			if float32(feeProtocol32&0xffff) != 0 {
				feeProtocol0 = 10000. / float32(feeProtocol32&0xffff)
			}

			if float32((feeProtocol32>>16)&0xffff) != 0 {
				feeProtocol1 = 10000. / float32((feeProtocol32>>16)&0xffff)
			}
		} else {
			return nil, fmt.Errorf("invalid feeProtocol: %v", values[5])
		}
	}

	unlocked, ok := values[6].(bool)
	if !ok {
		return nil, fmt.Errorf("invalid unlocked: %v", values[6])
	}

	return &bus.B_LP_V3_GetSlot0_Response{
		SqrtPriceX96: sqrtPriceX96,
		Tick:         tick.Int64(),
		FeeProtocol0: feeProtocol0,
		FeeProtocol1: feeProtocol1,
		Unlocked:     unlocked,
	}, nil
}
