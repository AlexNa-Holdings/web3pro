package price

import (
	"sort"
	"time"

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

var INIT_DELAY = 10 * time.Second
var PRICE_UPDATE_PERIOD time.Duration

func Init() {
	var err error

	PRICE_UPDATE_PERIOD, err = time.ParseDuration(cmn.Config.PriceUpdatePeriod)
	if err != nil {
		log.Error().Err(err).Msgf("Init: failed to parse price update period: %s", cmn.Config.PriceUpdatePeriod)
		return
	}

	go func() {
		time.Sleep(INIT_DELAY)
		for {
			if cmn.CurrentWallet != nil {
				Update(cmn.CurrentWallet)
				time.Sleep(PRICE_UPDATE_PERIOD)
			} else {
				time.Sleep(INIT_DELAY)
			}
		}
	}()
}

func GetPairs(chain_id uint, tokenAddr string) ([]Pair, error) {

	list, err := DSListPairs(chain_id, tokenAddr)
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
	n, err := DSUpdate(w)
	if err != nil {
		log.Error().Msgf("Update: failed to update from dexscreener: %v", err)
		return err
	}

	if n > 0 {

		err = w.Save()
		if err != nil {
			log.Error().Msgf("Update: failed to save wallet: %v", err)
		}

		cmn.Notify("Token prices updated")
	}

	return err
}
