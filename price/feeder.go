package price

import (
	"sort"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/rs/zerolog/log"
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

	list, err := DSListPairs(bchain, tokenAddr)
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

func Update(w *cmn.Wallet) error {
	err := DSUpdate(w)
	if err != nil {
		log.Error().Msgf("Update: failed to update from dexscreener: %v", err)
		return err
	}

	err = w.Save()
	if err != nil {
		log.Error().Msgf("Update: failed to save wallet: %v", err)
	}

	return err
}
