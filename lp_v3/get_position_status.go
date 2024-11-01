package lp_v3

import (
	"fmt"
	"math"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

var Q128, _ = new(big.Int).SetString("100000000000000000000000000000000", 16)
var Q256 = new(big.Int).Lsh(big.NewInt(1), 256)
var TWO96 = new(big.Int).Exp(big.NewInt(2), big.NewInt(96), nil)

func get_position_status(msg *bus.Message) (*bus.B_LP_V3_GetPositionStatus_Response, error) {
	req, ok := msg.Data.(*bus.B_LP_V3_GetPositionStatus)
	if !ok {
		return nil, fmt.Errorf("get_position_status: invalid data: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("get_position_status: no wallet")
	}

	p := w.GetLP_V3(req.ChainId, req.Provider)
	if p == nil {
		return nil, fmt.Errorf("get_position_status: no lp")
	}

	b := w.GetBlockchain(req.ChainId)
	if b == nil {
		return nil, fmt.Errorf("get_position_status: no blockchain")
	}

	lp := w.GetLP_V3Position(req.ChainId, req.Provider, req.NFT_Token)
	if lp == nil {
		return nil, fmt.Errorf("get_position_status: no lp")
	}

	nft_pos, slot0, fee_growth, tickLower, tickUpper, err := getV3PositionInfo(lp)
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf(">>>>>> nft_pos: NFT_Token=%s", lp.NFT_Token.String())

	amount0, amount1, in_range := calculateAmounts(nft_pos.Liquidity, slot0.SqrtPriceX96,
		getSqrtPriceX96FromTick(nft_pos.TickLower),
		getSqrtPriceX96FromTick(nft_pos.TickUpper))

	tokensOwed0, tokensOwed1 := calculateFees(fee_growth, nft_pos, slot0, tickLower, tickUpper)

	Gain0Dollars := 0.
	t0 := w.GetTokenByAddress(b.ChainId, nft_pos.Token0)
	if t0 != nil {
		Gain0Dollars = t0.Float64(tokensOwed0) * t0.Price
	}

	Gain1Dollars := 0.
	t1 := w.GetTokenByAddress(b.ChainId, nft_pos.Token1)
	if t1 != nil {
		Gain1Dollars = t1.Float64(tokensOwed1) * t1.Price
	}

	Liquidity0Dollars := 0.
	if t0 != nil {
		Liquidity0Dollars = t0.Float64(amount0) * t0.Price
	}

	Liquidity1Dollars := 0.
	if t1 != nil {
		Liquidity1Dollars = t1.Float64(amount1) * t1.Price
	}

	pn := fmt.Sprintf("%s@%s", p.Name, b.Currency)

	return &bus.B_LP_V3_GetPositionStatus_Response{
		On:                in_range,
		Token0:            nft_pos.Token0,
		Token1:            nft_pos.Token1,
		Liquidity0:        amount0,
		Liquidity1:        amount1,
		Liquidity0Dollars: Liquidity0Dollars,
		Liquidity1Dollars: Liquidity1Dollars,
		Gain0:             tokensOwed0,
		Gain1:             tokensOwed1,
		Gain0Dollars:      Gain0Dollars,
		Gain1Dollars:      Gain1Dollars,
		ProviderName:      pn,
		FeeProtocol0:      slot0.FeeProtocol0,
		FeeProtocol1:      slot0.FeeProtocol1,
		Owner:             lp.Owner,
		ChainId:           req.ChainId,
		Provider:          req.Provider,
	}, nil

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

	log.Debug().Msgf("getSqrtPriceX96FromTick: tick=%d, sqrtPriceX96=%s", tick, sqrtPriceX96.String())

	return sqrtPriceX96
}

func getV3PositionInfo(lp *cmn.LP_V3_Position) (
	*bus.B_LP_V3_GetNftPosition_Response,
	*bus.B_LP_V3_GetSlot0_Response,
	*bus.B_LP_V3_GetFeeGrowth_Response,
	*bus.B_LP_V3_GetTick_Response,
	*bus.B_LP_V3_GetTick_Response,
	error) {

	w := cmn.CurrentWallet
	if w == nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("no wallet")
	}

	b := w.GetBlockchain(lp.ChainId)
	if b == nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("no blockchain")
	}

	if b.Multicall != (common.Address{}) {
		return mc_getV3PositionInfo(lp)
	}

	nft_pos, err := _get_nft_position(lp.ChainId, lp.Provider, w.CurrentAddress, lp.NFT_Token)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	slot0, err := _get_slot0(lp.ChainId, lp.Pool)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	fee_growth, err := _get_fee_growth(lp.ChainId, lp.Pool)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	tickLower, err := _get_tick(lp.ChainId, lp.Pool, nft_pos.TickLower)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	tickUpper, err := _get_tick(lp.ChainId, lp.Pool, nft_pos.TickUpper)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return nft_pos, slot0, fee_growth, tickLower, tickUpper, nil
}

func calculateAmounts(liquidity, sqrtPriceX96, tickLowerSqrtPriceX96, tickUpperSqrtPriceX96 *big.Int) (*big.Int, *big.Int, bool) {
	in_range := false

	amount0 := big.NewInt(0)
	amount1 := big.NewInt(0)

	// Check if sqrtPriceX96 is within tickLower and tickUpper
	if sqrtPriceX96.Cmp(tickLowerSqrtPriceX96) <= 0 {
		// Price is below the range: Only token0 is involved
		amount0Numerator := new(big.Int).Sub(tickUpperSqrtPriceX96, tickLowerSqrtPriceX96)
		amount0Numerator.Mul(amount0Numerator, liquidity)

		// Keep precision high by multiplying first and dividing last
		denominator0 := new(big.Int).Mul(tickLowerSqrtPriceX96, tickUpperSqrtPriceX96)

		// Ensure numerator is multiplied by `2^96` to match precision
		amount0Numerator.Mul(amount0Numerator, TWO96)
		amount0.Div(amount0Numerator, denominator0)
	} else if sqrtPriceX96.Cmp(tickUpperSqrtPriceX96) >= 0 {
		// Price is above the range: Only token1 is involved
		numerator := new(big.Int).Sub(tickUpperSqrtPriceX96, tickLowerSqrtPriceX96)
		numerator.Mul(numerator, liquidity)
		amount1.Div(numerator, TWO96)
	} else {
		in_range = true
		// Price is within the range: Both tokens are involved

		// Calculate amount0
		amount0Numerator := new(big.Int).Sub(tickUpperSqrtPriceX96, sqrtPriceX96)
		amount0Numerator.Mul(amount0Numerator, liquidity)

		// Keep precision high by multiplying first and dividing last
		denominator0 := new(big.Int).Mul(sqrtPriceX96, tickUpperSqrtPriceX96)

		// Ensure numerator is multiplied by `2^96` to match precision
		amount0Numerator.Mul(amount0Numerator, TWO96)
		amount0.Div(amount0Numerator, denominator0)

		// Calculate amount1
		amount1Numerator := new(big.Int).Sub(sqrtPriceX96, tickLowerSqrtPriceX96)
		amount1Numerator.Mul(amount1Numerator, liquidity)

		amount1.Div(amount1Numerator, TWO96)
	}

	return amount0, amount1, in_range
}

// subIn256 handles overflows and underflows for 256-bit unsigned integers
func subIn256(x, y *big.Int) *big.Int {
	difference := new(big.Int).Sub(x, y)
	if difference.Sign() < 0 {
		return new(big.Int).Add(Q256, difference)
	}
	return difference
}

func getFeeGrowthInside(
	nft *bus.B_LP_V3_GetNftPosition_Response,
	growth *bus.B_LP_V3_GetFeeGrowth_Response,
	slot0 *bus.B_LP_V3_GetSlot0_Response,
	tickLower *bus.B_LP_V3_GetTick_Response,
	tickUpper *bus.B_LP_V3_GetTick_Response) (*big.Int, *big.Int) {

	// Calculate fee growth above for token0 and token1
	feeGrowthAbove0 := new(big.Int)
	feeGrowthAbove1 := new(big.Int)
	if slot0.Tick >= nft.TickUpper {
		log.Debug().Msgf("above: slot0.Tick >= nft.TickUpper")
		feeGrowthAbove0 = subIn256(growth.FeeGrowthGlobal0X128, tickUpper.FeeGrowthOutside0X128)
		feeGrowthAbove1 = subIn256(growth.FeeGrowthGlobal1X128, tickUpper.FeeGrowthOutside1X128)

	} else {
		log.Debug().Msgf("above: slot0.Tick < nft.TickUpper")
		feeGrowthAbove0.Set(tickUpper.FeeGrowthOutside0X128)
		feeGrowthAbove1.Set(tickUpper.FeeGrowthOutside1X128)
	}

	feeGrowthBelow0 := new(big.Int)
	feeGrowthBelow1 := new(big.Int)
	if slot0.Tick >= nft.TickLower {
		log.Debug().Msgf("below: slot0.Tick >= nft.TickLower")
		feeGrowthBelow0.Set(tickLower.FeeGrowthOutside0X128)
		feeGrowthBelow1.Set(tickLower.FeeGrowthOutside1X128)
	} else {
		log.Debug().Msgf("below: slot0.Tick < nft.TickLower")
		feeGrowthBelow0 = subIn256(growth.FeeGrowthGlobal0X128, tickLower.FeeGrowthOutside0X128)
		feeGrowthBelow1 = subIn256(growth.FeeGrowthGlobal1X128, tickLower.FeeGrowthOutside1X128)
	}

	// Calculate fee growth inside for token0 and token1
	feeGrowthInside0 := subIn256(growth.FeeGrowthGlobal0X128, feeGrowthBelow0)
	feeGrowthInside0 = subIn256(feeGrowthInside0, feeGrowthAbove0)

	feeGrowthInside1 := subIn256(growth.FeeGrowthGlobal1X128, feeGrowthBelow1)
	feeGrowthInside1 = subIn256(feeGrowthInside1, feeGrowthAbove1)

	return feeGrowthInside0, feeGrowthInside1
}

func calculateFees(growth *bus.B_LP_V3_GetFeeGrowth_Response,
	nft *bus.B_LP_V3_GetNftPosition_Response,
	slot0 *bus.B_LP_V3_GetSlot0_Response,
	tickLower *bus.B_LP_V3_GetTick_Response,
	tickUpper *bus.B_LP_V3_GetTick_Response) (*big.Int, *big.Int) {

	// Calculate fee growth inside for token0 and token1
	feeGrowthInside0, feeGrowthInside1 := getFeeGrowthInside(nft, growth, slot0, tickLower, tickUpper)

	// Calculate uncollected fees for token0 and token1
	uncollectedFees0 := new(big.Int).Sub(feeGrowthInside0, nft.FeeGrowthInside0LastX128)
	uncollectedFees1 := new(big.Int).Sub(feeGrowthInside1, nft.FeeGrowthInside1LastX128)

	uncollectedFees0.Mul(uncollectedFees0, nft.Liquidity)
	uncollectedFees1.Mul(uncollectedFees1, nft.Liquidity)

	// Adjust with liquidity scaling
	Q128 := new(big.Int).Lsh(big.NewInt(1), 128)
	uncollectedFees0 = uncollectedFees0.Div(uncollectedFees0, Q128)
	uncollectedFees1 = uncollectedFees1.Div(uncollectedFees1, Q128)

	return uncollectedFees0, uncollectedFees1
}

func mc_getV3PositionInfo(lp *cmn.LP_V3_Position) (
	*bus.B_LP_V3_GetNftPosition_Response,
	*bus.B_LP_V3_GetSlot0_Response,
	*bus.B_LP_V3_GetFeeGrowth_Response,
	*bus.B_LP_V3_GetTick_Response,
	*bus.B_LP_V3_GetTick_Response,
	error) {

	w := cmn.CurrentWallet
	if w == nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("no wallet")
	}

	data_positions, err := V3_MANAGER.Pack("positions", lp.NFT_Token)
	if err != nil {
		log.Error().Err(err).Msg("V3_ABI.Pack positions")
		return nil, nil, nil, nil, nil, err
	}

	data_slot0, err := V3_POOL_UNISWAP.Pack("slot0")
	if err != nil {
		log.Error().Err(err).Msg("V3_POOL_UNISWAP.Pack slot0")
		return nil, nil, nil, nil, nil, err
	}

	data_global0, err := V3_POOL_UNISWAP.Pack("feeGrowthGlobal0X128")
	if err != nil {
		log.Error().Err(err).Msg("V3_POOL_UNISWAP.Pack feeGrowthGlobal0X128")
		return nil, nil, nil, nil, nil, err
	}

	data_global1, err := V3_POOL_UNISWAP.Pack("feeGrowthGlobal1X128")
	if err != nil {
		log.Error().Err(err).Msg("V3_POOL_UNISWAP.Pack feeGrowthGlobal1X128")
		return nil, nil, nil, nil, nil, err
	}

	data_tick_lower, err := V3_POOL_UNISWAP.Pack("ticks", big.NewInt(lp.TickLower))
	if err != nil {
		log.Error().Err(err).Msg("V3_POOL_UNISWAP.Pack ticks")
		return nil, nil, nil, nil, nil, err
	}

	data_tick_upper, err := V3_POOL_UNISWAP.Pack("ticks", big.NewInt(lp.TickUpper))
	if err != nil {
		log.Error().Err(err).Msg("V3_POOL_UNISWAP.Pack ticks")
		return nil, nil, nil, nil, nil, err
	}

	// Use multicall
	calls := []bus.B_EthMultiCall_Call{
		{
			To:   lp.Provider,
			Data: data_positions,
		},
		{
			To:   lp.Pool,
			Data: data_slot0,
		},
		{
			To:   lp.Pool,
			Data: data_global0,
		},
		{
			To:   lp.Pool,
			Data: data_global1,
		},
		{
			To:   lp.Pool,
			Data: data_tick_lower,
		},
		{
			To:   lp.Pool,
			Data: data_tick_upper,
		},
	}

	resp := bus.Fetch("eth", "multi-call", &bus.B_EthMultiCall{
		ChainId: lp.ChainId,
		Calls:   calls,
	})

	if resp.Error != nil {
		log.Error().Err(resp.Error).Msg("multicall error")
		return nil, nil, nil, nil, nil, resp.Error
	}

	results := resp.Data.([][]byte)
	if len(results) != len(calls) {
		return nil, nil, nil, nil, nil, fmt.Errorf("unexpected number of results from multicall")
	}

	// Unpack all the results
	nft_pos, err := unpackNftPosition(results[0])
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	slot0, err := unpackSlot0(results[1])
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	feeGrowth, err := unpackFeeGrowth(results[2], results[3])
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	tickLower, err := unpackTick(results[4])
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	tickUpper, err := unpackTick(results[5])
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	return nft_pos, slot0, feeGrowth, tickLower, tickUpper, nil
}
