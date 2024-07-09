package price

import (
	"sort"
)

type Pair struct {
	PriceFeeder string
	PairAddress string
	BaseToken   string
	QuoteToken  string
	PriceUsd    float64
	Liquidity   float64
}

var KNOWN_FEEDERS = []string{"dexscreener"}

func GetPairs(bchain string, tokenAddr string) ([]Pair, error) {

	list, err := DSGetPairs(bchain, tokenAddr)
	if err != nil {
		return nil, err
	}

	sort.Slice(
		list,
		func(i, j int) bool {
			return list[i].Liquidity > list[j].Liquidity
		},
	)

	return list, nil
}
