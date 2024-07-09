package price

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

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

func DSGetPairs(bchain string, tokenAddr string) ([]Pair, error) {
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

	pairs := []Pair{}
	for _, pair := range response.Pairs {

		chain, err := extractBlockchainFromURL(pair.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to extract blockchain from URL: %w", err)
		}

		if chain != bchain {
			continue
		}

		price, err := strconv.ParseFloat(pair.PriceUsd, 64)
		if err != nil {
			log.Error().Err(err).Msgf("ParseFloat(%s) err: %v", pair.PriceUsd, err)
		}

		pairs = append(pairs,
			Pair{
				PriceFeeder: "dexscreener",
				PairAddress: pair.PairAddress,
				BaseToken:   pair.BaseToken.Symbol,
				QuoteToken:  pair.QuoteToken.Symbol,
				PriceUsd:    price,
				Liquidity:   pair.Liquidity.USD,
			})
	}

	return pairs, nil
}
