package price

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

var chain_names = map[int]string{
	1:    "ethereum",
	56:   "bsc",
	137:  "polygon",
	369:  "pulsechain",
	8453: "base",
}

type DSResponse struct {
	SchemaVersion string   `json:"schemaVersion"`
	Pairs         []DSPair `json:"pairs"`
}

type DSPair struct {
	URL           string       `json:"url"`
	PairAddress   string       `json:"pairAddress"`
	BaseToken     BaseToken    `json:"baseToken"`
	QuoteToken    QuoteToken   `json:"quoteToken"`
	PriceNative   string       `json:"priceNative"`
	PriceUsd      string       `json:"priceUsd"`
	Txns          Transactions `json:"txns"`
	Volume        Volume       `json:"volume"`
	PriceChange   PriceChange  `json:"priceChange"`
	Liquidity     Liquidity    `json:"liquidity"`
	FDV           float64      `json:"fdv"`
	PairCreatedAt int64        `json:"pairCreatedAt,omitempty"`
}

type BaseToken struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
}

type QuoteToken struct {
	Symbol string `json:"symbol"`
}

type Transactions struct {
	H24 TransactionDetails `json:"h24"`
	H6  TransactionDetails `json:"h6"`
	H1  TransactionDetails `json:"h1"`
	M5  TransactionDetails `json:"m5"`
}

type TransactionDetails struct {
	Buys  int `json:"buys"`
	Sells int `json:"sells"`
}

type Volume struct {
	H24 float64 `json:"h24"`
	H6  float64 `json:"h6"`
	H1  float64 `json:"h1"`
	M5  float64 `json:"m5"`
}

type PriceChange struct {
	H24 float64 `json:"h24"`
	H6  float64 `json:"h6"`
	H1  float64 `json:"h1"`
	M5  float64 `json:"m5"`
}

type Liquidity struct {
	USD   float64 `json:"usd"`
	Base  float64 `json:"base"`
	Quote float64 `json:"quote"`
}

func extractBlockchainFromURL(pairURL string) (string, error) {
	parsedURL, err := url.Parse(pairURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	parts := strings.Split(parsedURL.Path, "/")
	if len(parts) > 1 {
		return parts[1], nil
	}

	return "", fmt.Errorf("invalid URL format")
}

func DS_GetPriceInfoList(chain_id int, tokenAddr string) ([]PriceInfo, error) {

	chain_name, ok := chain_names[chain_id]
	if !ok {
		return nil, fmt.Errorf("unknown chain id: %d", chain_id)
	}

	url := fmt.Sprintf("https://api.dexscreener.com/latest/dex/tokens/%s", tokenAddr)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK HTTP status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response DSResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}

	pairs := []PriceInfo{}
	for _, pair := range response.Pairs {

		chain, err := extractBlockchainFromURL(pair.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to extract blockchain from URL: %w", err)
		}

		if chain != chain_name {
			continue
		}

		price, err := strconv.ParseFloat(pair.PriceUsd, 64)
		if err != nil {
			log.Error().Err(err).Msgf("ParseFloat(%s) err: %v", pair.PriceUsd, err)
		}

		pairs = append(pairs,
			PriceInfo{
				PriceFeeder: "dexscreener",
				PairID:      pair.PairAddress,
				BaseToken:   pair.BaseToken.Symbol,
				QuoteToken:  pair.QuoteToken.Symbol,
				PriceUsd:    price,
				Liquidity:   pair.Liquidity.USD,
				URL:         "https://dexscreener.com/" + chain + "/" + pair.PairAddress,
			})
	}

	return pairs, nil
}

func DSGetPairs(chain_id int, pairList string) ([]PriceInfo, error) {

	chain_name, ok := chain_names[chain_id]
	if !ok {
		return nil, fmt.Errorf("unknown chain id: %d", chain_id)
	}

	url := fmt.Sprintf("https://api.dexscreener.com/latest/dex/pairs/%s/%s", chain_name, pairList)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK HTTP status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response DSResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}

	pairs := []PriceInfo{}
	for _, pair := range response.Pairs {

		chain, err := extractBlockchainFromURL(pair.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to extract blockchain from URL: %w", err)
		}

		if chain != chain_name {
			continue
		}

		price, err := strconv.ParseFloat(pair.PriceUsd, 64)
		if err != nil {
			log.Error().Err(err).Msgf("ParseFloat(%s) err: %v", pair.PriceUsd, err)
		}

		pairs = append(pairs,
			PriceInfo{
				PriceFeeder:   "dexscreener",
				PairID:        pair.PairAddress,
				BaseToken:     pair.BaseToken.Symbol,
				QuoteToken:    pair.QuoteToken.Symbol,
				PriceUsd:      price,
				PriceChange24: pair.PriceChange.H24,
				Liquidity:     pair.Liquidity.USD,
			})
	}

	return pairs, nil
}

func DS_Update(w *cmn.Wallet) (int, error) { // number of pairs updated
	n_updated := 0

	for _, b := range w.Blockchains {

		if chain_names[b.ChainId] == "" {
			continue
		}
		tokens_to_update := []*cmn.Token{}
		for _, t := range w.Tokens {
			if t.ChainId == b.ChainId && t.PriceFeeder == "dexscreener" && t.PriceFeedParam != "" {
				tokens_to_update = append(tokens_to_update, t)
			}
		}

		pair_list := ""
		added := make(map[string]bool)
		for _, t := range tokens_to_update {

			if added[t.PriceFeedParam] {
				continue
			}
			added[t.PriceFeedParam] = true

			if pair_list != "" {
				pair_list += ","
			}
			pair_list += t.PriceFeedParam
		}

		if len(pair_list) > 0 {
			pairs, err := DSGetPairs(b.ChainId, pair_list)
			if err != nil {
				log.Error().Err(err).Msgf("DSUpdate: failed to get pairs from dexscreener: %v", err)
				return 0, fmt.Errorf("DSUpdate: failed to get pairs from dexscreener: %w", err)
			}

			for i, p := range pairs {

				for _, t := range tokens_to_update {
					b := w.GetBlockchain(t.ChainId)
					if b == nil {
						continue
					}

					if t.PriceFeedParam == p.PairID {
						t.Price = pairs[i].PriceUsd
						t.PriceChange24 = pairs[i].PriceChange24
						t.PriceTimestamp = time.Now()
						n_updated++

						if t.Native {
							// update wrapped token price if needed
							if b.WTokenAddress != (common.Address{}) {
								wrapped_t := w.GetTokenByAddress(b.ChainId, b.WTokenAddress)
								if wrapped_t != nil && wrapped_t.PriceFeedParam == "" {
									wrapped_t.Price = p.PriceUsd
									wrapped_t.PriceChange24 = p.PriceChange24
									wrapped_t.PriceTimestamp = time.Now()
									n_updated++
								}
							}
						} else {
							// update native token price if needed
							if b.WTokenAddress.Cmp(t.Address) == 0 {
								native_t, err := w.GetNativeToken(b)
								if err != nil && native_t != nil && native_t.PriceFeedParam == "" {
									native_t.Price = p.PriceUsd
									native_t.PriceChange24 = p.PriceChange24
									native_t.PriceTimestamp = time.Now()
									n_updated++
								}
							}
						}

					}
				}
			}
		}
	}
	return n_updated, nil
}
