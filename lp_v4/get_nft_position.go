package lp_v4

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

func getNftPosition(msg *bus.Message) (*bus.B_LP_V4_GetNftPosition_Response, error) {
	req, ok := msg.Data.(*bus.B_LP_V4_GetNftPosition)
	if !ok {
		return nil, fmt.Errorf("get_nft_position: invalid data: %v", msg.Data)
	}

	return _getNftPosition(req.ChainId, req.Provider, req.From, req.NFT_Token)
}

func _getNftPosition(chainId int,
	provider common.Address,
	from common.Address,
	nftToken *big.Int) (*bus.B_LP_V4_GetNftPosition_Response, error) {

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("get_nft_position: no wallet")
	}

	b := w.GetBlockchain(chainId)
	if b == nil {
		return nil, fmt.Errorf("get_nft_position: blockchain not found")
	}

	// Try to use multicall to batch RPC calls
	if b.Multicall != (common.Address{}) {
		return getNftPositionViaMulticall(chainId, provider, from, nftToken)
	}

	// Fallback to individual calls if multicall not available
	return getNftPositionIndividual(chainId, provider, from, nftToken)
}

// getNftPositionIndividual makes individual RPC calls (fallback when multicall unavailable)
func getNftPositionIndividual(chainId int,
	provider common.Address,
	from common.Address,
	nftToken *big.Int) (*bus.B_LP_V4_GetNftPosition_Response, error) {

	// Call positionInfo to get poolId, tickLower, tickUpper
	data, err := V4_POSITION_MANAGER.Pack("positionInfo", nftToken)
	if err != nil {
		log.Error().Err(err).Msg("V4_POSITION_MANAGER.Pack positionInfo")
		return nil, err
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: chainId,
		To:      provider,
		From:    from,
		Data:    data,
	})

	if resp.Error != nil {
		log.Error().Err(resp.Error).Msg("eth call positionInfo")
		return nil, resp.Error
	}

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		log.Error().Err(err).Msg("hexutil.Decode positionInfo")
		return nil, err
	}

	// Unpack positionInfo result: returns a packed uint256 (PositionInfo)
	// Bit layout (from LSB to MSB):
	// - bits 0-7: hasSubscriber (8 bits)
	// - bits 8-31: tickLower (24 bits, signed)
	// - bits 32-55: tickUpper (24 bits, signed)
	// - bits 56-255: poolId (200 bits, truncated to bytes25)
	var posInfo *big.Int

	err = V4_POSITION_MANAGER.UnpackIntoInterface(&posInfo, "positionInfo", output)
	if err != nil {
		log.Error().Err(err).Msg("V4_POSITION_MANAGER.UnpackIntoInterface positionInfo")
		return nil, err
	}

	// Extract tickLower (bits 8-31, 24-bit signed int)
	tickLowerRaw := new(big.Int).Rsh(posInfo, 8)
	tickLowerRaw.And(tickLowerRaw, big.NewInt(0xFFFFFF))
	tickLower := signExtend24(tickLowerRaw.Int64())

	// Extract tickUpper (bits 32-55, 24-bit signed int)
	tickUpperRaw := new(big.Int).Rsh(posInfo, 32)
	tickUpperRaw.And(tickUpperRaw, big.NewInt(0xFFFFFF))
	tickUpper := signExtend24(tickUpperRaw.Int64())

	// Extract poolId (bits 56-255, 200 bits = 25 bytes)
	poolIdBig := new(big.Int).Rsh(posInfo, 56)
	poolIdBytes := poolIdBig.Bytes()

	// Create bytes25 for poolKeys lookup (right-padded to 25 bytes)
	var poolIdBytes25 [25]byte
	if len(poolIdBytes) <= 25 {
		copy(poolIdBytes25[25-len(poolIdBytes):], poolIdBytes)
	} else {
		copy(poolIdBytes25[:], poolIdBytes[len(poolIdBytes)-25:])
	}

	// Also keep a bytes32 version for compatibility
	var poolId [32]byte
	copy(poolId[32-len(poolIdBytes):], poolIdBytes)

	log.Debug().Msgf("positionInfo: tickLower=%d, tickUpper=%d, poolId=%x, poolIdBytes25=%x", tickLower, tickUpper, poolId, poolIdBytes25)

	// Query poolKeys to get currency0, currency1, fee, tickSpacing, hooks
	data, err = V4_POSITION_MANAGER.Pack("poolKeys", poolIdBytes25)
	if err != nil {
		log.Error().Err(err).Msg("V4_POSITION_MANAGER.Pack poolKeys")
		return nil, err
	}

	resp = bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: chainId,
		To:      provider,
		From:    from,
		Data:    data,
	})

	if resp.Error != nil {
		log.Error().Err(resp.Error).Msg("eth call poolKeys")
		return nil, resp.Error
	}

	output, err = hexutil.Decode(resp.Data.(string))
	if err != nil {
		log.Error().Err(err).Msg("hexutil.Decode poolKeys")
		return nil, err
	}

	// Unpack poolKeys result: (currency0, currency1, fee, tickSpacing, hooks)
	var currency0, currency1, hookAddress common.Address
	var fee, tickSpacing *big.Int

	err = V4_POSITION_MANAGER.UnpackIntoInterface(
		&[]interface{}{&currency0, &currency1, &fee, &tickSpacing, &hookAddress},
		"poolKeys", output)
	if err != nil {
		log.Error().Err(err).Msg("V4_POSITION_MANAGER.UnpackIntoInterface poolKeys")
		return nil, err
	}

	log.Debug().Msgf("poolKeys: currency0=%s, currency1=%s, fee=%d, tickSpacing=%d, hooks=%s",
		currency0.Hex(), currency1.Hex(), fee.Int64(), tickSpacing.Int64(), hookAddress.Hex())

	// Now get liquidity using getPositionLiquidity
	data, err = V4_POSITION_MANAGER.Pack("getPositionLiquidity", nftToken)
	if err != nil {
		log.Error().Err(err).Msg("V4_POSITION_MANAGER.Pack getPositionLiquidity")
		return nil, err
	}

	resp = bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: chainId,
		To:      provider,
		From:    from,
		Data:    data,
	})

	if resp.Error != nil {
		log.Error().Err(resp.Error).Msg("eth call getPositionLiquidity")
		return nil, resp.Error
	}

	output, err = hexutil.Decode(resp.Data.(string))
	if err != nil {
		log.Error().Err(err).Msg("hexutil.Decode getPositionLiquidity")
		return nil, err
	}

	var liquidity *big.Int
	err = V4_POSITION_MANAGER.UnpackIntoInterface(&liquidity, "getPositionLiquidity", output)
	if err != nil {
		log.Error().Err(err).Msg("V4_POSITION_MANAGER.UnpackIntoInterface getPositionLiquidity")
		return nil, err
	}

	return &bus.B_LP_V4_GetNftPosition_Response{
		PoolId:      poolId,
		TickLower:   tickLower,
		TickUpper:   tickUpper,
		Liquidity:   liquidity,
		Currency0:   currency0,
		Currency1:   currency1,
		Fee:         fee.Int64(),
		TickSpacing: tickSpacing.Int64(),
		HookAddress: hookAddress,
	}, nil
}

// getNftPositionViaMulticall batches positionInfo and getPositionLiquidity into a single multicall
// Note: poolKeys requires the result from positionInfo, so we do 2 multicalls:
// 1. First multicall: positionInfo + getPositionLiquidity
// 2. Second call: poolKeys (needs poolIdBytes25 from positionInfo)
func getNftPositionViaMulticall(chainId int,
	provider common.Address,
	from common.Address,
	nftToken *big.Int) (*bus.B_LP_V4_GetNftPosition_Response, error) {

	// Pack positionInfo call
	positionInfoData, err := V4_POSITION_MANAGER.Pack("positionInfo", nftToken)
	if err != nil {
		log.Error().Err(err).Msg("V4_POSITION_MANAGER.Pack positionInfo")
		return nil, err
	}

	// Pack getPositionLiquidity call
	liquidityData, err := V4_POSITION_MANAGER.Pack("getPositionLiquidity", nftToken)
	if err != nil {
		log.Error().Err(err).Msg("V4_POSITION_MANAGER.Pack getPositionLiquidity")
		return nil, err
	}

	// First multicall: positionInfo + getPositionLiquidity
	calls := []bus.B_EthMultiCall_Call{
		{To: provider, Data: positionInfoData}, // 0: positionInfo
		{To: provider, Data: liquidityData},    // 1: getPositionLiquidity
	}

	resp := bus.Fetch("eth", "multi-call", &bus.B_EthMultiCall{
		ChainId: chainId,
		From:    from,
		Calls:   calls,
	})

	if resp.Error != nil {
		log.Warn().Err(resp.Error).Msg("Multicall failed for V4 NFT position, falling back to individual calls")
		return getNftPositionIndividual(chainId, provider, from, nftToken)
	}

	results, ok := resp.Data.([][]byte)
	if !ok || len(results) < 2 {
		log.Warn().Msg("Invalid multicall response for V4 NFT position, falling back to individual calls")
		return getNftPositionIndividual(chainId, provider, from, nftToken)
	}

	// Unpack positionInfo (result 0)
	var posInfo *big.Int
	err = V4_POSITION_MANAGER.UnpackIntoInterface(&posInfo, "positionInfo", results[0])
	if err != nil {
		log.Error().Err(err).Msg("V4_POSITION_MANAGER.UnpackIntoInterface positionInfo from multicall")
		return nil, err
	}

	// Extract tickLower (bits 8-31, 24-bit signed int)
	tickLowerRaw := new(big.Int).Rsh(posInfo, 8)
	tickLowerRaw.And(tickLowerRaw, big.NewInt(0xFFFFFF))
	tickLower := signExtend24(tickLowerRaw.Int64())

	// Extract tickUpper (bits 32-55, 24-bit signed int)
	tickUpperRaw := new(big.Int).Rsh(posInfo, 32)
	tickUpperRaw.And(tickUpperRaw, big.NewInt(0xFFFFFF))
	tickUpper := signExtend24(tickUpperRaw.Int64())

	// Extract poolId (bits 56-255, 200 bits = 25 bytes)
	poolIdBig := new(big.Int).Rsh(posInfo, 56)
	poolIdBytes := poolIdBig.Bytes()

	// Create bytes25 for poolKeys lookup
	var poolIdBytes25 [25]byte
	if len(poolIdBytes) <= 25 {
		copy(poolIdBytes25[25-len(poolIdBytes):], poolIdBytes)
	} else {
		copy(poolIdBytes25[:], poolIdBytes[len(poolIdBytes)-25:])
	}

	// Also keep a bytes32 version for compatibility
	var poolId [32]byte
	copy(poolId[32-len(poolIdBytes):], poolIdBytes)

	// Unpack getPositionLiquidity (result 1)
	var liquidity *big.Int
	err = V4_POSITION_MANAGER.UnpackIntoInterface(&liquidity, "getPositionLiquidity", results[1])
	if err != nil {
		log.Error().Err(err).Msg("V4_POSITION_MANAGER.UnpackIntoInterface getPositionLiquidity from multicall")
		return nil, err
	}

	log.Debug().Msgf("multicall positionInfo: tickLower=%d, tickUpper=%d, poolId=%x, liquidity=%s",
		tickLower, tickUpper, poolId, liquidity.String())

	// Second call: poolKeys (needs poolIdBytes25 from positionInfo)
	poolKeysData, err := V4_POSITION_MANAGER.Pack("poolKeys", poolIdBytes25)
	if err != nil {
		log.Error().Err(err).Msg("V4_POSITION_MANAGER.Pack poolKeys")
		return nil, err
	}

	resp = bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: chainId,
		To:      provider,
		From:    from,
		Data:    poolKeysData,
	})

	if resp.Error != nil {
		log.Error().Err(resp.Error).Msg("eth call poolKeys")
		return nil, resp.Error
	}

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		log.Error().Err(err).Msg("hexutil.Decode poolKeys")
		return nil, err
	}

	// Unpack poolKeys result: (currency0, currency1, fee, tickSpacing, hooks)
	var currency0, currency1, hookAddress common.Address
	var fee, tickSpacing *big.Int

	err = V4_POSITION_MANAGER.UnpackIntoInterface(
		&[]interface{}{&currency0, &currency1, &fee, &tickSpacing, &hookAddress},
		"poolKeys", output)
	if err != nil {
		log.Error().Err(err).Msg("V4_POSITION_MANAGER.UnpackIntoInterface poolKeys")
		return nil, err
	}

	log.Debug().Msgf("poolKeys: currency0=%s, currency1=%s, fee=%d, tickSpacing=%d, hooks=%s",
		currency0.Hex(), currency1.Hex(), fee.Int64(), tickSpacing.Int64(), hookAddress.Hex())

	return &bus.B_LP_V4_GetNftPosition_Response{
		PoolId:      poolId,
		TickLower:   tickLower,
		TickUpper:   tickUpper,
		Liquidity:   liquidity,
		Currency0:   currency0,
		Currency1:   currency1,
		Fee:         fee.Int64(),
		TickSpacing: tickSpacing.Int64(),
		HookAddress: hookAddress,
	}, nil
}

// signExtend24 sign-extends a 24-bit value to int64
func signExtend24(val int64) int64 {
	if val&0x800000 != 0 {
		// Negative number, extend the sign
		return val | ^int64(0xFFFFFF)
	}
	return val
}
