package price

import (
	"sort"
	"time"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/rs/zerolog/log"
)

type PriceInfo struct {
	PriceFeeder   string
	PairID        string
	BaseToken     string
	QuoteToken    string
	PriceUsd      float64
	PriceChange24 float64
	Liquidity     float64
	URL           string
}

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

func GetPriceInfoList(chain_id int, tokenAddr string) ([]PriceInfo, error) {

	ds_list, err := DS_GetPriceInfoList(chain_id, tokenAddr)
	if err != nil {
		return nil, err
	}

	var cmc_list []PriceInfo
	if cmn.Config.CMC_API_KEY != "" {
		cmc_list, err = CMC_GetPriceInfoList(chain_id, tokenAddr)
		if err != nil {
			log.Debug().Msgf("GetPriceInfoList: failed to get price info from CoinMarketCap: %v", err)
		}
	}

	list := append(ds_list, cmc_list...)

	sort.Slice(
		list,
		func(i, j int) bool {
			return list[i].Liquidity > list[j].Liquidity
		},
	)

	return list, nil
}

func Update(w *cmn.Wallet) error {
	n_ds, err := DS_Update(w)
	if err != nil {
		log.Error().Msgf("Update: failed to update from dexscreener: %v", err)
	}

	n_cmc, err := CMC_Update(w)
	if err != nil {
		log.Error().Msgf("Update: failed to update from CoinMarketCap: %v", err)
	}

	if n_ds+n_cmc > 0 {
		err = w.Save()
		if err != nil {
			log.Error().Msgf("Update: failed to save wallet: %v", err)
		}

		bus.Send("price", "updated", nil)
		bus.Send("ui", "notify", "Token prices updated")
	}

	return err
}
