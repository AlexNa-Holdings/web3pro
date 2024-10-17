package lp_v3

import (
	"bytes"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog/log"
)

func get_pool_position(msg *bus.Message) (*bus.B_LP_V3_GetPoolPosition_Response, error) {
	req, ok := msg.Data.(*bus.B_LP_V3_GetPoolPosition)
	if !ok {
		return nil, fmt.Errorf("get_position: invalid data: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("get_position: no wallet")
	}

	key, err := generatePositionKey(req.Provider, req.TickLower, req.TickUpper)
	if err != nil {
		log.Error().Err(err).Msg("generatePositionKey")
		return nil, err
	}

	data, err := V3_POOL_UNISWAP.Pack("positions", key)
	if err != nil {
		log.Error().Err(err).Msg("V3_ABI.Pack positions")
		return nil, err
	}

	log.Debug().Msgf("pool: %v", req.Pool)
	log.Debug().Msgf("provider: %v", req.Provider)
	log.Debug().Msgf("tickLower: %v", req.TickLower)
	log.Debug().Msgf("tickUpper: %v", req.TickUpper)
	log.Debug().Msgf("key: %v", hexutil.Encode(key[:]))

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: req.ChainId,
		To:      req.Pool,
		From:    req.Provider,
		Data:    data,
	})

	if resp.Error != nil {
		log.Error().Err(resp.Error).Msg("eth call")
		return nil, resp.Error
	}

	var r_data bus.B_LP_V3_GetPoolPosition_Response

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		log.Error().Err(err).Msg("hexutil.Decode")
		return nil, err
	}
	err = V3_POOL_UNISWAP.UnpackIntoInterface(
		&[]interface{}{
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

	return &r_data, nil
}

// Function to pack int24 values as 3-byte slices
func int24ToBytes(value int64) ([]byte, error) {
	// Ensure the value fits into int24 (-2^23 to 2^23-1)
	if value < -8388608 || value > 8388607 {
		return nil, fmt.Errorf("value out of int24 range")
	}

	// Create a 3-byte slice
	buf := make([]byte, 3)

	// Write the last 3 bytes of the int32 value into the 3-byte slice
	buf[0] = byte(value >> 16)
	buf[1] = byte(value >> 8)
	buf[2] = byte(value)

	return buf, nil
}

// Function to generate the keccak256 hash key as a common.Hash ([32]byte)
func generatePositionKey(owner common.Address, tickLower, tickUpper int64) (common.Hash, error) {
	// Convert int24 values to byte slices
	tickLowerBytes, err := int24ToBytes(tickLower)
	if err != nil {
		return common.Hash{}, err
	}

	tickUpperBytes, err := int24ToBytes(tickUpper)
	if err != nil {
		return common.Hash{}, err
	}

	// Create a buffer and concatenate the address and tick values
	buf := new(bytes.Buffer)
	buf.Write(owner.Bytes())  // Write the owner's address
	buf.Write(tickLowerBytes) // Write tickLower as a 3-byte value
	buf.Write(tickUpperBytes) // Write tickUpper as a 3-byte value

	log.Debug().Msgf("buf: %v", hexutil.Encode(buf.Bytes()))

	// Hash the buffer using Keccak256
	key := crypto.Keccak256Hash(buf.Bytes())

	return key, nil
}
