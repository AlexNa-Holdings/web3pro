package price

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
)

const CMC_API = "https://pro-api.coinmarketcap.com/v1"

// CMCMapResponse holds the response from the cryptocurrency/map endpoint
type CMCMapResponse struct {
	Status Status   `json:"status"`
	Data   []Crypto `json:"data"`
}

// CMCQuotesResponse holds the response from the cryptocurrency/quotes/latest endpoint
type CMCQuotesResponse struct {
	Status struct {
		Timestamp    string `json:"timestamp"`
		ErrorCode    int    `json:"error_code"`
		ErrorMessage string `json:"error_message,omitempty"`
		Elapsed      int    `json:"elapsed"`
		CreditCount  int    `json:"credit_count"`
	} `json:"status"`
	Data map[string]struct {
		ID                int      `json:"id"`
		Name              string   `json:"name"`
		Symbol            string   `json:"symbol"`
		Slug              string   `json:"slug"`
		CirculatingSupply float64  `json:"circulating_supply"`
		TotalSupply       float64  `json:"total_supply"`
		MaxSupply         float64  `json:"max_supply"`
		LastUpdated       string   `json:"last_updated"`
		DateAdded         string   `json:"date_added"`
		NumMarketPairs    int      `json:"num_market_pairs"`
		Tags              []string `json:"tags"`
		Platform          struct {
			ID           int    `json:"id"`
			Name         string `json:"name"`
			Symbol       string `json:"symbol"`
			Slug         string `json:"slug"`
			TokenAddress string `json:"token_address"`
		} `json:"platform,omitempty"`
		Quote map[string]struct {
			Price            float64 `json:"price"`
			Volume24h        float64 `json:"volume_24h"`
			VolumeChange24h  float64 `json:"volume_change_24h"`
			PercentChange1h  float64 `json:"percent_change_1h"`
			PercentChange24h float64 `json:"percent_change_24h"`
			PercentChange7d  float64 `json:"percent_change_7d"`
			PercentChange30d float64 `json:"percent_change_30d"`
			PercentChange60d float64 `json:"percent_change_60d"`
			PercentChange90d float64 `json:"percent_change_90d"`
			MarketCap        float64 `json:"market_cap"`
			LastUpdated      string  `json:"last_updated"`
		} `json:"quote"`
	} `json:"data"`
}

// Status holds metadata about the response
type Status struct {
	Timestamp    string `json:"timestamp"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Elapsed      int    `json:"elapsed"`
	CreditCount  int    `json:"credit_count"`
}

// Crypto holds information about each cryptocurrency
type Crypto struct {
	ID                  int    `json:"id"`
	Name                string `json:"name"`
	Symbol              string `json:"symbol"`
	Slug                string `json:"slug"`
	Rank                int    `json:"rank"`
	IsActive            int    `json:"is_active"`
	FirstHistoricalData string `json:"first_historical_data"`
	LastHistoricalData  string `json:"last_historical_data"`
}

func CMC_GetPriceInfoList(chain_id int, tokenAddr string) ([]PriceInfo, error) {
	if cmn.Config.CMC_API_KEY == "" {
		return nil, fmt.Errorf("CMC API key not set")
	}

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("no wallet open")
	}

	b := w.GetBlockchain(chain_id)
	if b == nil {
		return nil, fmt.Errorf("blockchain not found")
	}

	t := w.GetToken(chain_id, tokenAddr)
	if t == nil {
		return nil, fmt.Errorf("token not found")
	}

	slags := []string{}

	// treat some known tokens
	if b.ChainId == 1 && t.Native {
		slags = append(slags, "ethereum")
	} else {

		url := fmt.Sprintf("%s/cryptocurrency/map?symbol=%s&sort=cmc_rank", CMC_API, t.Symbol)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create GET request: %w", err)
		}

		req.Header.Add("X-CMC_PRO_API_KEY", cmn.Config.CMC_API_KEY)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make GET request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("received non-OK HTTP status: %s", resp.Status)
		}

		var response CMCMapResponse

		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
		}

		// create the list of slugs
		for _, crypto := range response.Data {
			slags = append(slags, crypto.Slug)
		}
	}

	// get the latest prices
	return CMC_GetLatest(slags)
}

func CMC_GetLatest(slags []string) ([]PriceInfo, error) {
	// Create a new slice of PriceInfo structs
	pairs := []PriceInfo{}

	if len(slags) == 0 {
		return pairs, nil
	}

	if cmn.Config.CMC_API_KEY == "" {
		return nil, fmt.Errorf("CMC API key not set")
	}

	slags_str := strings.Join(slags, ",")

	url := fmt.Sprintf("%s/cryptocurrency/quotes/latest?slug=%s", CMC_API, slags_str)

	// Create a new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}

	// Add the X-CMC_PRO_API_KEY header
	req.Header.Add("X-CMC_PRO_API_KEY", cmn.Config.CMC_API_KEY)

	// Create a new client
	client := &http.Client{}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %w", err)
	}

	// Close the response body when the function returns
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Create a new CMCResponse struct
	var response CMCQuotesResponse

	// Unmarshal the JSON response into the CMCResponse struct
	if err_um := json.Unmarshal(body, &response); err_um != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}

	// Check if the response status code is not OK
	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("received non-OK HTTP status: %s", resp.Status+" "+response.Status.ErrorMessage)
	}

	// Iterate over the data in the response
	for _, data := range response.Data {
		// Iterate over the quote data
		for _, quote := range data.Quote {
			// Append a new PriceInfo struct to the pairs slice
			pairs = append(pairs,
				PriceInfo{
					PriceFeeder:   "coinmarketcap",
					PairID:        data.Slug,
					BaseToken:     "",
					QuoteToken:    "",
					PriceUsd:      quote.Price,
					PriceChange24: quote.PercentChange24h,
					Liquidity:     quote.Volume24h,
					URL:           fmt.Sprintf("https://coinmarketcap.com/currencies/%s/", data.Slug),
				})
		}
	}

	// Return the pairs slice and nil error
	return pairs, nil
}

func CMC_Update(w *cmn.Wallet) (int, error) { // number of pairs updated
	n_updated := 0

	slags := []string{}
	already_added := map[string]bool{}

	for _, t := range w.Tokens {
		if t.PriceFeeder != "coinmarketcap" || t.PriceFeedParam == "" {
			continue
		}

		if !already_added[t.PriceFeedParam] {
			already_added[t.PriceFeedParam] = true
			slags = append(slags, t.PriceFeedParam)
		}
	}

	pi_list, err := CMC_GetLatest(slags)
	if err != nil {
		return n_updated, err
	}

	for _, pi := range pi_list {
		for _, t := range w.Tokens {
			if t.PriceFeeder != "coinmarketcap" || t.PriceFeedParam != pi.PairID {
				continue
			}

			b := w.GetBlockchain(t.ChainId)
			if b == nil {
				continue
			}

			// Update this token's price
			t.Price = pi.PriceUsd
			t.PriceChange24 = pi.PriceChange24
			t.PriceTimestamp = time.Now()
			n_updated++

			// Sync wrapped/native token price if they don't have their own feeder
			if t.Native {
				// This is native token - update wrapped token price if needed
				if b.WTokenAddress != (common.Address{}) {
					wrapped_t := w.GetTokenByAddress(t.ChainId, b.WTokenAddress)
					if wrapped_t != nil && wrapped_t.PriceFeedParam == "" {
						wrapped_t.Price = pi.PriceUsd
						wrapped_t.PriceChange24 = pi.PriceChange24
						wrapped_t.PriceTimestamp = time.Now()
						n_updated++
					}
				}
			} else if t.Address.Cmp(b.WTokenAddress) == 0 {
				// This is wrapped token - update native token price if needed
				native_t, _ := w.GetNativeToken(b)
				if native_t != nil && native_t.PriceFeedParam == "" {
					native_t.Price = pi.PriceUsd
					native_t.PriceChange24 = pi.PriceChange24
					native_t.PriceTimestamp = time.Now()
					n_updated++
				}
			}
		}
	}

	return n_updated, nil
}
