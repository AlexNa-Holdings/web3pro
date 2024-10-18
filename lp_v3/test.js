const ZERO = JSBI.BigInt(0);
const Q128 = JSBI.exponentiate(JSBI.BigInt(2), JSBI.BigInt(128));
const Q256 = JSBI.exponentiate(JSBI.BigInt(2), JSBI.BigInt(256));

// this handles the over and underflows which is needed for all subtraction in the fees math
function subIn256(x, y){
  const difference = JSBI.subtract(x, y)
  if (JSBI.lessThan(difference, ZERO)) {
    return JSBI.add(Q256, difference)
  } else {
    return difference}}

async function getFees(feeGrowthGlobal0, feeGrowthGlobal1, feeGrowth0Low, feeGrowth0Hi, feeGrowthInside0, feeGrowth1Low, feeGrowth1Hi, feeGrowthInside1, liquidity, decimals0, decimals1, tickLower, tickUpper, tickCurrent){
                                // all needs to be bigNumber
	let feeGrowthGlobal_0 = toBigNumber(feeGrowthGlobal0);
	let feeGrowthGlobal_1 = toBigNumber(feeGrowthGlobal1);
	let tickLowerFeeGrowthOutside_0 = toBigNumber(feeGrowth0Low);
	let tickLowerFeeGrowthOutside_1 = toBigNumber(feeGrowth1Low);
	let tickUpperFeeGrowthOutside_0 = toBigNumber(feeGrowth0Hi);
	let tickUpperFeeGrowthOutside_1 = toBigNumber(feeGrowth1Hi);
                               // preset variables to 0 BigNumber
	let tickLowerFeeGrowthBelow_0 = ZERO;
	let tickLowerFeeGrowthBelow_1 = ZERO;
	let tickUpperFeeGrowthAbove_0 = ZERO;
	let tickUpperFeeGrowthAbove_1 = ZERO;

              // As stated above there is different math needed if the position is in or out of range
                              // If current tick is above the range fg- fo,iu Growth Above range

	if (tickCurrent >= tickUpper){
		tickUpperFeeGrowthAbove_0 = subIn256(feeGrowthGlobal_0, tickUpperFeeGrowthOutside_0);
		tickUpperFeeGrowthAbove_1 = subIn256(feeGrowthGlobal_1, tickUpperFeeGrowthOutside_1);
	} else{                // Else if current tick is in range only need fg for upper growth
		tickUpperFeeGrowthAbove_0 = tickUpperFeeGrowthOutside_0
		tickUpperFeeGrowthAbove_1 = tickUpperFeeGrowthOutside_1
	}
                              // If current tick is in range  only need fg for lower growth
	if (tickCurrent >= tickLower){
		tickLowerFeeGrowthBelow_0 = tickLowerFeeGrowthOutside_0
		tickLowerFeeGrowthBelow_1 = tickLowerFeeGrowthOutside_1
	} else{                // If current tick is above the range fg- fo,il Growth below range
		tickLowerFeeGrowthBelow_0 = subIn256(feeGrowthGlobal_0, tickLowerFeeGrowthOutside_0);
		tickLowerFeeGrowthBelow_1 = subIn256(feeGrowthGlobal_1, tickLowerFeeGrowthOutside_1);
	}

                             //   fr(t1) For both token0 and token1
	let fr_t1_0 = subIn256(subIn256(feeGrowthGlobal_0, tickLowerFeeGrowthBelow_0), tickUpperFeeGrowthAbove_0);
	let fr_t1_1 = subIn256(subIn256(feeGrowthGlobal_1, tickLowerFeeGrowthBelow_1), tickUpperFeeGrowthAbove_1);
                                // feeGrowthInside to BigNumber
	let feeGrowthInsideLast_0 = toBigNumber(feeGrowthInside0);
	let feeGrowthInsideLast_1 = toBigNumber(feeGrowthInside1);

	// The final calculations uncollected fees formula
	// for both token 0 and token 1 since we now know everything that is needed to compute it
       // subtracting the two values and then multiplying with liquidity l *(fr(t1) - fr(t0)) 
	let uncollectedFees_0 = (liquidity * subIn256(fr_t1_0, feeGrowthInsideLast_0)) / Q128;
	let uncollectedFees_1 = (liquidity * subIn256(fr_t1_1, feeGrowthInsideLast_1)) / Q128;

	console.log("Amount fees token 0 in lowest decimal: "+Math.floor(uncollectedFees_0));
	console.log("Amount fees token 1 in lowest decimal: "+Math.floor(uncollectedFees_1));

	// Decimal adjustment to get final results
	let uncollectedFeesAdjusted_0 = (uncollectedFees_0 / toBigNumber(10**decimals0)).toFixed(decimals0);
	let uncollectedFeesAdjusted_1 = (uncollectedFees_1 / toBigNumber(10**decimals1)).toFixed(decimals1);
	console.log("Amount fees token 0 Human format: "+uncollectedFeesAdjusted_0);
	console.log("Amount fees token 1 Human format: "+uncollectedFeesAdjusted_1);
}

// Output position 1
Amount fees token 0 in lowest decimal: 84250230863135890
Amount fees token 1 in lowest decimal: 661007116889360
Amount fees token 0 Human format: 0.084250230863135891
Amount fees token 1 Human format: 0.000661007116889361