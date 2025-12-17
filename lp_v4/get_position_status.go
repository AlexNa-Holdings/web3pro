package lp_v4

import (
	"fmt"
	"math"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog/log"
)

var TWO96 = new(big.Int).Exp(big.NewInt(2), big.NewInt(96), nil)
var Q128 = new(big.Int).Lsh(big.NewInt(1), 128)
var Q256 = new(big.Int).Lsh(big.NewInt(1), 256)

func getPositionStatus(msg *bus.Message) (*bus.B_LP_V4_GetPositionStatus_Response, error) {
	req, ok := msg.Data.(*bus.B_LP_V4_GetPositionStatus)
	if !ok {
		return nil, fmt.Errorf("invalid request: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("no wallet")
	}

	pos := w.GetLP_V4Position(req.ChainId, req.Provider, req.NFT_Token)
	if pos == nil {
		return nil, fmt.Errorf("position not found")
	}

	lp := w.GetLP_V4(req.ChainId, req.Provider)
	if lp == nil {
		return nil, fmt.Errorf("provider not found")
	}

	b := w.GetBlockchain(pos.ChainId)
	if b == nil {
		return nil, fmt.Errorf("blockchain not found")
	}

	// Always fetch fresh position data from chain (uses multicall internally if available)
	freshData, err := _getNftPosition(pos.ChainId, pos.Provider, pos.Owner, pos.NFT_Token)
	if err != nil {
		log.Error().Err(err).Msg("V4 getPositionStatus: failed to fetch fresh position data")
		return nil, fmt.Errorf("failed to fetch position data: %w", err)
	}

	// Use fresh data
	tickSpacing := freshData.TickSpacing
	currency0 := freshData.Currency0
	currency1 := freshData.Currency1
	fee := freshData.Fee
	hookAddress := freshData.HookAddress
	liquidity := freshData.Liquidity
	tickLower := freshData.TickLower
	tickUpper := freshData.TickUpper

	log.Debug().Msgf("V4 getPositionStatus: NFT=%s, Currency0=%s, Currency1=%s, Fee=%d, TickSpacing=%d, Hook=%s",
		req.NFT_Token.String(), currency0.Hex(), currency1.Hex(), fee, tickSpacing, hookAddress.Hex())
	log.Debug().Msgf("V4 getPositionStatus: TickLower=%d, TickUpper=%d, Liquidity=%s, StateView=%s",
		tickLower, tickUpper, liquidity.String(), lp.StateView.Hex())

	// Compute the actual poolId (keccak256 hash of PoolKey) for StateView
	// The pos.PoolId is the truncated 25-byte version used by PositionManager
	actualPoolId := computePoolId(currency0, currency1, fee, tickSpacing, hookAddress)

	// Initialize variables
	sqrtPriceX96 := big.NewInt(0)
	currentTick := int64(0)
	gain0 := big.NewInt(0)
	gain1 := big.NewInt(0)

	// Try to use multicall to batch all StateView RPC calls into one
	if b.Multicall != (common.Address{}) && lp.StateView != (common.Address{}) {
		sqrtPriceX96, currentTick, gain0, gain1 = getPositionStatusViaMulticall(
			pos.ChainId, lp.StateView, actualPoolId, lp.Provider,
			pos.NFT_Token, tickLower, tickUpper, liquidity,
		)
	} else {
		// Fallback to individual calls if multicall not available
		var err error
		sqrtPriceX96, currentTick, err = getSlot0(pos.ChainId, lp.StateView, actualPoolId)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to get slot0, using zero values")
			sqrtPriceX96 = big.NewInt(0)
			currentTick = 0
		}

		if lp.StateView != (common.Address{}) && liquidity != nil && liquidity.Cmp(big.NewInt(0)) > 0 {
			gain0, gain1 = calculateV4Fees(pos.ChainId, lp.StateView, actualPoolId, lp.Provider,
				pos.NFT_Token, tickLower, tickUpper, currentTick, liquidity)
		}
	}

	log.Debug().Msgf("V4 getPositionStatus: sqrtPriceX96=%s, currentTick=%d", sqrtPriceX96.String(), currentTick)

	// Calculate whether position is in range
	on := currentTick >= tickLower && currentTick < tickUpper

	// Calculate liquidity amounts for each token
	liquidity0 := big.NewInt(0)
	liquidity1 := big.NewInt(0)
	if liquidity != nil && liquidity.Cmp(big.NewInt(0)) > 0 && sqrtPriceX96.Cmp(big.NewInt(0)) > 0 {
		liquidity0, liquidity1, _ = calculateAmounts(
			liquidity,
			sqrtPriceX96,
			getSqrtPriceX96FromTick(tickLower),
			getSqrtPriceX96FromTick(tickUpper),
		)
		log.Debug().Msgf("V4 getPositionStatus: liquidity0=%s, liquidity1=%s", liquidity0.String(), liquidity1.String())
	} else {
		log.Warn().Msgf("V4 getPositionStatus: skipping calculateAmounts - Liquidity=%v, sqrtPriceX96=%s",
			liquidity, sqrtPriceX96.String())
	}

	// Calculate dollar values
	liquidity0Dollars := 0.0
	liquidity1Dollars := 0.0
	gain0Dollars := 0.0
	gain1Dollars := 0.0
	t0 := w.GetTokenByAddress(pos.ChainId, currency0)
	t1 := w.GetTokenByAddress(pos.ChainId, currency1)

	if t0 != nil {
		liquidity0Dollars = t0.Float64(liquidity0) * t0.Price
		gain0Dollars = t0.Float64(gain0) * t0.Price
	}
	if t1 != nil {
		liquidity1Dollars = t1.Float64(liquidity1) * t1.Price
		gain1Dollars = t1.Float64(gain1) * t1.Price
	}

	pn := fmt.Sprintf("%s@%s", lp.Name, b.GetShortName())

	return &bus.B_LP_V4_GetPositionStatus_Response{
		Owner:             pos.Owner,
		ChainId:           pos.ChainId,
		NFT_Token:         pos.NFT_Token,
		Currency0:         currency0,
		Currency1:         currency1,
		Provider:          pos.Provider,
		PoolManager:       pos.PoolManager,
		PoolId:            pos.PoolId,
		TickLower:         tickLower,
		TickUpper:         tickUpper,
		On:                on,
		Fee:               fee,
		Liquidity:         liquidity,
		Liquidity0:        liquidity0,
		Liquidity1:        liquidity1,
		Liquidity0Dollars: liquidity0Dollars,
		Liquidity1Dollars: liquidity1Dollars,
		Gain0:             gain0,
		Gain1:             gain1,
		Gain0Dollars:      gain0Dollars,
		Gain1Dollars:      gain1Dollars,
		ProviderName:      pn,
		HookAddress:       hookAddress,
	}, nil
}

func getSlot0(chainId int, stateView common.Address, poolId [32]byte) (*big.Int, int64, error) {
	// Skip if StateView is not configured
	if stateView == (common.Address{}) {
		return nil, 0, fmt.Errorf("StateView not configured")
	}

	// Pack the getSlot0 call
	data, err := V4_STATE_VIEW.Pack("getSlot0", poolId)
	if err != nil {
		return nil, 0, fmt.Errorf("pack getSlot0: %w", err)
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: chainId,
		To:      stateView,
		Data:    data,
	})

	if resp.Error != nil {
		return nil, 0, resp.Error
	}

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		return nil, 0, fmt.Errorf("decode response: %w", err)
	}

	log.Debug().Msgf("getSlot0 response length: %d bytes, data: %x", len(output), output)

	// Check if response is empty or too short
	if len(output) < 64 {
		return nil, 0, fmt.Errorf("getSlot0 response too short: %d bytes (pool may not exist)", len(output))
	}

	// Unpack: (uint160 sqrtPriceX96, int24 tick, uint24 protocolFee, uint24 lpFee)
	result, err := V4_STATE_VIEW.Unpack("getSlot0", output)
	if err != nil {
		return nil, 0, fmt.Errorf("unpack getSlot0: %w", err)
	}

	if len(result) < 2 {
		return nil, 0, fmt.Errorf("getSlot0 returned insufficient values: %d", len(result))
	}

	sqrtPriceX96, ok := result[0].(*big.Int)
	if !ok {
		return nil, 0, fmt.Errorf("getSlot0: sqrtPriceX96 is not *big.Int: %T", result[0])
	}

	tick, ok := result[1].(*big.Int)
	if !ok {
		return nil, 0, fmt.Errorf("getSlot0: tick is not *big.Int: %T", result[1])
	}

	log.Debug().Msgf("getSlot0 result: sqrtPriceX96=%s, tick=%d", sqrtPriceX96.String(), tick.Int64())

	return sqrtPriceX96, tick.Int64(), nil
}

func getSqrtPriceX96FromTick(tick int64) *big.Int {
	// Calculate 1.0001^tick as a float
	price := math.Pow(1.0001, math.Abs(float64(tick)))

	// If tick is negative, invert the price
	if tick < 0 {
		price = 1 / price
	}

	// Take the square root of the price
	sqrtPrice := math.Sqrt(price)

	// Multiply by 2^96 to convert to Q96 format
	two96 := new(big.Float).SetInt(TWO96)
	sqrtPriceX96Float := new(big.Float).Mul(big.NewFloat(sqrtPrice), two96)

	// Convert to *big.Int
	sqrtPriceX96 := new(big.Int)
	sqrtPriceX96Float.Int(sqrtPriceX96)

	return sqrtPriceX96
}

func calculateAmounts(liquidity, sqrtPriceX96, tickLowerSqrtPriceX96, tickUpperSqrtPriceX96 *big.Int) (*big.Int, *big.Int, bool) {
	inRange := false

	amount0 := big.NewInt(0)
	amount1 := big.NewInt(0)

	// Check if sqrtPriceX96 is within tickLower and tickUpper
	if sqrtPriceX96.Cmp(tickLowerSqrtPriceX96) <= 0 {
		// Price is below the range: Only token0 is involved
		amount0Numerator := new(big.Int).Sub(tickUpperSqrtPriceX96, tickLowerSqrtPriceX96)
		amount0Numerator.Mul(amount0Numerator, liquidity)

		denominator0 := new(big.Int).Mul(tickLowerSqrtPriceX96, tickUpperSqrtPriceX96)

		amount0Numerator.Mul(amount0Numerator, TWO96)
		amount0.Div(amount0Numerator, denominator0)
	} else if sqrtPriceX96.Cmp(tickUpperSqrtPriceX96) >= 0 {
		// Price is above the range: Only token1 is involved
		numerator := new(big.Int).Sub(tickUpperSqrtPriceX96, tickLowerSqrtPriceX96)
		numerator.Mul(numerator, liquidity)
		amount1.Div(numerator, TWO96)
	} else {
		inRange = true
		// Price is within the range: Both tokens are involved

		// Calculate amount0
		amount0Numerator := new(big.Int).Sub(tickUpperSqrtPriceX96, sqrtPriceX96)
		amount0Numerator.Mul(amount0Numerator, liquidity)

		denominator0 := new(big.Int).Mul(sqrtPriceX96, tickUpperSqrtPriceX96)

		amount0Numerator.Mul(amount0Numerator, TWO96)
		amount0.Div(amount0Numerator, denominator0)

		// Calculate amount1
		amount1Numerator := new(big.Int).Sub(sqrtPriceX96, tickLowerSqrtPriceX96)
		amount1Numerator.Mul(amount1Numerator, liquidity)

		amount1.Div(amount1Numerator, TWO96)
	}

	return amount0, amount1, inRange
}

// computePoolId computes the actual poolId (keccak256 hash of PoolKey) used by PoolManager/StateView
// PoolKey = (Currency currency0, Currency currency1, uint24 fee, int24 tickSpacing, IHooks hooks)
func computePoolId(currency0, currency1 common.Address, fee, tickSpacing int64, hooks common.Address) [32]byte {
	// ABI encode the PoolKey struct
	// PoolKey is: (address, address, uint24, int24, address)
	addressType, _ := abi.NewType("address", "", nil)
	uint24Type, _ := abi.NewType("uint24", "", nil)
	int24Type, _ := abi.NewType("int24", "", nil)

	arguments := abi.Arguments{
		{Type: addressType},
		{Type: addressType},
		{Type: uint24Type},
		{Type: int24Type},
		{Type: addressType},
	}

	encoded, err := arguments.Pack(currency0, currency1, big.NewInt(fee), big.NewInt(tickSpacing), hooks)
	if err != nil {
		log.Error().Err(err).Msg("Failed to encode PoolKey")
		return [32]byte{}
	}

	// Compute keccak256 hash
	hash := crypto.Keccak256(encoded)
	var poolId [32]byte
	copy(poolId[:], hash)

	log.Debug().Msgf("computePoolId: currency0=%s, currency1=%s, fee=%d, tickSpacing=%d, hooks=%s -> poolId=%x",
		currency0.Hex(), currency1.Hex(), fee, tickSpacing, hooks.Hex(), poolId)

	return poolId
}

// calculateV4Fees calculates uncollected fees for a V4 position
func calculateV4Fees(chainId int, stateView common.Address, poolId [32]byte, provider common.Address,
	nftToken *big.Int, tickLower, tickUpper, currentTick int64, liquidity *big.Int) (*big.Int, *big.Int) {

	// Get fee growth globals
	feeGrowthGlobal0, feeGrowthGlobal1, err := getFeeGrowthGlobals(chainId, stateView, poolId)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get fee growth globals")
		return big.NewInt(0), big.NewInt(0)
	}

	// Get tick info for lower and upper ticks
	tickLowerInfo, err := getTickInfo(chainId, stateView, poolId, tickLower)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get lower tick info")
		return big.NewInt(0), big.NewInt(0)
	}

	tickUpperInfo, err := getTickInfo(chainId, stateView, poolId, tickUpper)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get upper tick info")
		return big.NewInt(0), big.NewInt(0)
	}

	// Get position info with feeGrowthInsideLast values
	feeGrowthInside0Last, feeGrowthInside1Last, err := getPositionFeeGrowth(chainId, stateView, poolId, provider, nftToken, tickLower, tickUpper)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get position fee growth")
		return big.NewInt(0), big.NewInt(0)
	}

	// Calculate fee growth inside
	feeGrowthInside0, feeGrowthInside1 := getFeeGrowthInsideV4(
		currentTick, tickLower, tickUpper,
		feeGrowthGlobal0, feeGrowthGlobal1,
		tickLowerInfo.feeGrowthOutside0, tickLowerInfo.feeGrowthOutside1,
		tickUpperInfo.feeGrowthOutside0, tickUpperInfo.feeGrowthOutside1,
	)

	// Calculate uncollected fees
	uncollectedFees0 := subIn256(feeGrowthInside0, feeGrowthInside0Last)
	uncollectedFees1 := subIn256(feeGrowthInside1, feeGrowthInside1Last)

	uncollectedFees0.Mul(uncollectedFees0, liquidity)
	uncollectedFees1.Mul(uncollectedFees1, liquidity)

	uncollectedFees0.Div(uncollectedFees0, Q128)
	uncollectedFees1.Div(uncollectedFees1, Q128)

	log.Debug().Msgf("V4 fees: gain0=%s, gain1=%s", uncollectedFees0.String(), uncollectedFees1.String())

	return uncollectedFees0, uncollectedFees1
}

type tickInfo struct {
	feeGrowthOutside0 *big.Int
	feeGrowthOutside1 *big.Int
}

// getFeeGrowthGlobals fetches global fee growth from StateView
func getFeeGrowthGlobals(chainId int, stateView common.Address, poolId [32]byte) (*big.Int, *big.Int, error) {
	data, err := V4_STATE_VIEW.Pack("getFeeGrowthGlobals", poolId)
	if err != nil {
		return nil, nil, fmt.Errorf("pack getFeeGrowthGlobals: %w", err)
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: chainId,
		To:      stateView,
		Data:    data,
	})

	if resp.Error != nil {
		return nil, nil, resp.Error
	}

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		return nil, nil, fmt.Errorf("decode response: %w", err)
	}

	result, err := V4_STATE_VIEW.Unpack("getFeeGrowthGlobals", output)
	if err != nil {
		return nil, nil, fmt.Errorf("unpack getFeeGrowthGlobals: %w", err)
	}

	if len(result) < 2 {
		return nil, nil, fmt.Errorf("getFeeGrowthGlobals returned insufficient values")
	}

	feeGrowth0 := result[0].(*big.Int)
	feeGrowth1 := result[1].(*big.Int)

	return feeGrowth0, feeGrowth1, nil
}

// getTickInfo fetches tick info from StateView
func getTickInfo(chainId int, stateView common.Address, poolId [32]byte, tick int64) (*tickInfo, error) {
	data, err := V4_STATE_VIEW.Pack("getTickInfo", poolId, big.NewInt(tick))
	if err != nil {
		return nil, fmt.Errorf("pack getTickInfo: %w", err)
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: chainId,
		To:      stateView,
		Data:    data,
	})

	if resp.Error != nil {
		return nil, resp.Error
	}

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	result, err := V4_STATE_VIEW.Unpack("getTickInfo", output)
	if err != nil {
		return nil, fmt.Errorf("unpack getTickInfo: %w", err)
	}

	// Returns: liquidityGross, liquidityNet, feeGrowthOutside0X128, feeGrowthOutside1X128
	if len(result) < 4 {
		return nil, fmt.Errorf("getTickInfo returned insufficient values")
	}

	return &tickInfo{
		feeGrowthOutside0: result[2].(*big.Int),
		feeGrowthOutside1: result[3].(*big.Int),
	}, nil
}

// getPositionFeeGrowth fetches position's feeGrowthInsideLast values from StateView
func getPositionFeeGrowth(chainId int, stateView common.Address, poolId [32]byte,
	provider common.Address, nftToken *big.Int, tickLower, tickUpper int64) (*big.Int, *big.Int, error) {

	// For V4, the position owner in PoolManager is the PositionManager contract
	// The salt is derived from the NFT token ID
	// salt = keccak256(abi.encodePacked(msg.sender, tokenId))
	// But for getPositionInfo, we need the actual position key

	// The position in V4 PoolManager is keyed by:
	// - owner: the PositionManager contract address
	// - tickLower
	// - tickUpper
	// - salt: bytes32 derived from the NFT owner and tokenId

	// For now, try with provider as owner and tokenId as salt
	var salt [32]byte
	tokenBytes := nftToken.Bytes()
	copy(salt[32-len(tokenBytes):], tokenBytes)

	data, err := V4_STATE_VIEW.Pack("getPositionInfo", poolId, provider, big.NewInt(tickLower), big.NewInt(tickUpper), salt)
	if err != nil {
		return nil, nil, fmt.Errorf("pack getPositionInfo: %w", err)
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: chainId,
		To:      stateView,
		Data:    data,
	})

	if resp.Error != nil {
		return nil, nil, resp.Error
	}

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		return nil, nil, fmt.Errorf("decode response: %w", err)
	}

	result, err := V4_STATE_VIEW.Unpack("getPositionInfo", output)
	if err != nil {
		return nil, nil, fmt.Errorf("unpack getPositionInfo: %w", err)
	}

	// Returns: liquidity, feeGrowthInside0LastX128, feeGrowthInside1LastX128
	if len(result) < 3 {
		return nil, nil, fmt.Errorf("getPositionInfo returned insufficient values")
	}

	feeGrowthInside0Last := result[1].(*big.Int)
	feeGrowthInside1Last := result[2].(*big.Int)

	return feeGrowthInside0Last, feeGrowthInside1Last, nil
}

// getFeeGrowthInsideV4 calculates fee growth inside the position's tick range
func getFeeGrowthInsideV4(currentTick, tickLower, tickUpper int64,
	feeGrowthGlobal0, feeGrowthGlobal1 *big.Int,
	tickLowerFeeGrowthOutside0, tickLowerFeeGrowthOutside1 *big.Int,
	tickUpperFeeGrowthOutside0, tickUpperFeeGrowthOutside1 *big.Int) (*big.Int, *big.Int) {

	// Calculate fee growth above for token0 and token1
	feeGrowthAbove0 := new(big.Int)
	feeGrowthAbove1 := new(big.Int)
	if currentTick >= tickUpper {
		feeGrowthAbove0 = subIn256(feeGrowthGlobal0, tickUpperFeeGrowthOutside0)
		feeGrowthAbove1 = subIn256(feeGrowthGlobal1, tickUpperFeeGrowthOutside1)
	} else {
		feeGrowthAbove0.Set(tickUpperFeeGrowthOutside0)
		feeGrowthAbove1.Set(tickUpperFeeGrowthOutside1)
	}

	// Calculate fee growth below for token0 and token1
	feeGrowthBelow0 := new(big.Int)
	feeGrowthBelow1 := new(big.Int)
	if currentTick >= tickLower {
		feeGrowthBelow0.Set(tickLowerFeeGrowthOutside0)
		feeGrowthBelow1.Set(tickLowerFeeGrowthOutside1)
	} else {
		feeGrowthBelow0 = subIn256(feeGrowthGlobal0, tickLowerFeeGrowthOutside0)
		feeGrowthBelow1 = subIn256(feeGrowthGlobal1, tickLowerFeeGrowthOutside1)
	}

	// Calculate fee growth inside for token0 and token1
	feeGrowthInside0 := subIn256(feeGrowthGlobal0, feeGrowthBelow0)
	feeGrowthInside0 = subIn256(feeGrowthInside0, feeGrowthAbove0)

	feeGrowthInside1 := subIn256(feeGrowthGlobal1, feeGrowthBelow1)
	feeGrowthInside1 = subIn256(feeGrowthInside1, feeGrowthAbove1)

	return feeGrowthInside0, feeGrowthInside1
}

// subIn256 handles overflows and underflows for 256-bit unsigned integers
func subIn256(x, y *big.Int) *big.Int {
	difference := new(big.Int).Sub(x, y)
	if difference.Sign() < 0 {
		return new(big.Int).Add(Q256, difference)
	}
	return difference
}

// getPositionStatusViaMulticall batches all StateView RPC calls into a single multicall
// to reduce RPC calls from 5 to 1, avoiding rate limiting issues
// Calls batched: getSlot0, getFeeGrowthGlobals, getTickInfo (lower), getTickInfo (upper), getPositionInfo
func getPositionStatusViaMulticall(
	chainId int,
	stateView common.Address,
	poolId [32]byte,
	provider common.Address,
	nftToken *big.Int,
	tickLower, tickUpper int64,
	liquidity *big.Int,
) (sqrtPriceX96 *big.Int, currentTick int64, gain0, gain1 *big.Int) {
	// Initialize with zero values
	sqrtPriceX96 = big.NewInt(0)
	currentTick = 0
	gain0 = big.NewInt(0)
	gain1 = big.NewInt(0)

	// Pack all call data
	slot0Data, err := V4_STATE_VIEW.Pack("getSlot0", poolId)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to pack getSlot0 for multicall")
		return
	}

	feeGrowthGlobalsData, err := V4_STATE_VIEW.Pack("getFeeGrowthGlobals", poolId)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to pack getFeeGrowthGlobals for multicall")
		return
	}

	tickLowerData, err := V4_STATE_VIEW.Pack("getTickInfo", poolId, big.NewInt(tickLower))
	if err != nil {
		log.Warn().Err(err).Msg("Failed to pack getTickInfo (lower) for multicall")
		return
	}

	tickUpperData, err := V4_STATE_VIEW.Pack("getTickInfo", poolId, big.NewInt(tickUpper))
	if err != nil {
		log.Warn().Err(err).Msg("Failed to pack getTickInfo (upper) for multicall")
		return
	}

	// Build salt for getPositionInfo
	var salt [32]byte
	tokenBytes := nftToken.Bytes()
	copy(salt[32-len(tokenBytes):], tokenBytes)

	positionInfoData, err := V4_STATE_VIEW.Pack("getPositionInfo", poolId, provider, big.NewInt(tickLower), big.NewInt(tickUpper), salt)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to pack getPositionInfo for multicall")
		return
	}

	// Build multicall request
	calls := []bus.B_EthMultiCall_Call{
		{To: stateView, Data: slot0Data},           // 0: getSlot0
		{To: stateView, Data: feeGrowthGlobalsData}, // 1: getFeeGrowthGlobals
		{To: stateView, Data: tickLowerData},        // 2: getTickInfo (lower)
		{To: stateView, Data: tickUpperData},        // 3: getTickInfo (upper)
		{To: stateView, Data: positionInfoData},     // 4: getPositionInfo
	}

	resp := bus.Fetch("eth", "multi-call", &bus.B_EthMultiCall{
		ChainId: chainId,
		Calls:   calls,
	})

	if resp.Error != nil {
		log.Warn().Err(resp.Error).Msg("Multicall failed for LP V4 position status")
		bus.Send("ui", "notify-error", fmt.Sprintf("LP V4 multicall failed: %v", resp.Error))
		return
	}

	results, ok := resp.Data.([][]byte)
	if !ok || len(results) < 5 {
		log.Warn().Msg("Invalid multicall response for LP V4 position status")
		bus.Send("ui", "notify-error", "LP V4: Invalid multicall response")
		return
	}

	// Unpack getSlot0 (result 0)
	if len(results[0]) >= 64 {
		result, err := V4_STATE_VIEW.Unpack("getSlot0", results[0])
		if err == nil && len(result) >= 2 {
			if sqrtPrice, ok := result[0].(*big.Int); ok && sqrtPrice != nil {
				sqrtPriceX96 = sqrtPrice
			}
			if tick, ok := result[1].(*big.Int); ok && tick != nil {
				currentTick = tick.Int64()
			}
		}
	}

	// If no liquidity, no need to calculate fees
	if liquidity == nil || liquidity.Cmp(big.NewInt(0)) <= 0 {
		return
	}

	// Unpack getFeeGrowthGlobals (result 1)
	feeGrowthGlobal0 := big.NewInt(0)
	feeGrowthGlobal1 := big.NewInt(0)
	if len(results[1]) >= 64 {
		result, err := V4_STATE_VIEW.Unpack("getFeeGrowthGlobals", results[1])
		if err == nil && len(result) >= 2 {
			if fg0, ok := result[0].(*big.Int); ok && fg0 != nil {
				feeGrowthGlobal0 = fg0
			}
			if fg1, ok := result[1].(*big.Int); ok && fg1 != nil {
				feeGrowthGlobal1 = fg1
			}
		}
	}

	// Unpack getTickInfo (lower) (result 2)
	tickLowerFeeGrowthOutside0 := big.NewInt(0)
	tickLowerFeeGrowthOutside1 := big.NewInt(0)
	if len(results[2]) >= 64 {
		result, err := V4_STATE_VIEW.Unpack("getTickInfo", results[2])
		if err == nil && len(result) >= 4 {
			if fg0, ok := result[2].(*big.Int); ok && fg0 != nil {
				tickLowerFeeGrowthOutside0 = fg0
			}
			if fg1, ok := result[3].(*big.Int); ok && fg1 != nil {
				tickLowerFeeGrowthOutside1 = fg1
			}
		}
	}

	// Unpack getTickInfo (upper) (result 3)
	tickUpperFeeGrowthOutside0 := big.NewInt(0)
	tickUpperFeeGrowthOutside1 := big.NewInt(0)
	if len(results[3]) >= 64 {
		result, err := V4_STATE_VIEW.Unpack("getTickInfo", results[3])
		if err == nil && len(result) >= 4 {
			if fg0, ok := result[2].(*big.Int); ok && fg0 != nil {
				tickUpperFeeGrowthOutside0 = fg0
			}
			if fg1, ok := result[3].(*big.Int); ok && fg1 != nil {
				tickUpperFeeGrowthOutside1 = fg1
			}
		}
	}

	// Unpack getPositionInfo (result 4)
	feeGrowthInside0Last := big.NewInt(0)
	feeGrowthInside1Last := big.NewInt(0)
	if len(results[4]) >= 64 {
		result, err := V4_STATE_VIEW.Unpack("getPositionInfo", results[4])
		if err == nil && len(result) >= 3 {
			if fg0, ok := result[1].(*big.Int); ok && fg0 != nil {
				feeGrowthInside0Last = fg0
			}
			if fg1, ok := result[2].(*big.Int); ok && fg1 != nil {
				feeGrowthInside1Last = fg1
			}
		}
	}

	// Calculate fee growth inside
	feeGrowthInside0, feeGrowthInside1 := getFeeGrowthInsideV4(
		currentTick, tickLower, tickUpper,
		feeGrowthGlobal0, feeGrowthGlobal1,
		tickLowerFeeGrowthOutside0, tickLowerFeeGrowthOutside1,
		tickUpperFeeGrowthOutside0, tickUpperFeeGrowthOutside1,
	)

	// Calculate uncollected fees
	gain0 = subIn256(feeGrowthInside0, feeGrowthInside0Last)
	gain1 = subIn256(feeGrowthInside1, feeGrowthInside1Last)

	gain0.Mul(gain0, liquidity)
	gain1.Mul(gain1, liquidity)

	gain0.Div(gain0, Q128)
	gain1.Div(gain1, Q128)

	log.Debug().Msgf("V4 multicall fees: gain0=%s, gain1=%s", gain0.String(), gain1.String())

	return
}
