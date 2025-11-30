package lp_v4

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

// GraphQL response structures for Uniswap V4 subgraph
type SubgraphResponse struct {
	Data   SubgraphData    `json:"data"`
	Errors []SubgraphError `json:"errors,omitempty"`
}

type SubgraphError struct {
	Message string `json:"message"`
}

type SubgraphData struct {
	Positions []SubgraphPosition `json:"positions"`
}

// Position in V4 subgraph - basic NFT position info
type SubgraphPosition struct {
	ID                 string `json:"id"`
	TokenId            string `json:"tokenId"`
	Owner              string `json:"owner"`
	Origin             string `json:"origin"`
	CreatedAtTimestamp string `json:"createdAtTimestamp"`
}

// Pool type for V4 subgraph
type SubgraphPool struct {
	ID      string        `json:"id"`
	Token0  SubgraphToken `json:"token0"`
	Token1  SubgraphToken `json:"token1"`
	FeeTier string        `json:"feeTier"`
	Hooks   string        `json:"hooks"`
}

type SubgraphToken struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
}

// Response for pool query
type SubgraphPoolResponse struct {
	Data   SubgraphPoolData `json:"data"`
	Errors []SubgraphError  `json:"errors,omitempty"`
}

type SubgraphPoolData struct {
	Pool *SubgraphPool `json:"pool"`
}

func discover(msg *bus.Message) error {
	found := 0

	req, ok := msg.Data.(bus.B_LP_V4_Discover)
	if !ok {
		return fmt.Errorf("invalid request: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return fmt.Errorf("discover: no wallet")
	}

	chain_id := 0
	b := w.GetBlockchain(req.ChainId)
	if b != nil {
		chain_id = b.ChainId
	}

	name := req.Name

	ui.Printf("Discovering LP v4\n\n")

	log.Debug().Msgf("LP_V4 Providers count: %d", len(w.LP_V4_Providers))
	if len(w.LP_V4_Providers) == 0 {
		ui.Printf("No LP v4 providers configured. Use 'lp_v4 add' to add a provider first.\n")
		return nil
	}

	for _, pl := range w.LP_V4_Providers {
		log.Debug().Msgf("Checking provider: %s, ChainId: %d, SubgraphURL: %s", pl.Name, pl.ChainId, pl.SubgraphURL)
		if name != "" && pl.Name != name {
			continue
		}

		if chain_id != 0 && pl.ChainId != chain_id {
			continue
		}

		b := w.GetBlockchain(pl.ChainId)
		if b == nil {
			log.Error().Msgf("discover: Blockchain not found: %d", pl.ChainId)
			continue
		}

		if pl.SubgraphURL == "" {
			ui.Printf("Skipping %s %s: no subgraph URL configured\n", b.Name, pl.Name)
			continue
		}

		ui.Printf("Discovering LP v4: %s %s\n", b.Name, pl.Name)

		for _, addr := range w.Addresses {
			ui.Printf("  ")
			cmn.AddAddressShortLink(ui.Terminal.Screen, addr.Address)
			ui.Printf(" %s \n", addr.Name)
			ui.Flush()

			log.Debug().Msgf("Querying subgraph for address: %s", addr.Address.Hex())
			positions, err := querySubgraph(pl.SubgraphURL, addr.Address)
			if err != nil {
				log.Error().Err(err).Msg("querySubgraph")
				ui.PrintErrorf("Subgraph query failed: %v\n", err)
				continue
			}
			log.Debug().Msgf("Found %d positions for address %s", len(positions), addr.Address.Hex())

			for _, pos := range positions {
				tokenId, ok := new(big.Int).SetString(pos.TokenId, 10)
				if !ok {
					log.Error().Msgf("Invalid tokenId: %s", pos.TokenId)
					continue
				}

				// Fetch on-chain position details
				posResp := msg.Fetch("lp_v4", "get-nft-position", &bus.B_LP_V4_GetNftPosition{
					ChainId:   pl.ChainId,
					Provider:  pl.Provider,
					From:      addr.Address,
					NFT_Token: tokenId,
				})

				if posResp.Error != nil {
					log.Error().Err(posResp.Error).Msgf("get-nft-position for token %s", tokenId.String())
					ui.PrintErrorf("    Failed to get position info: %v\n", posResp.Error)
					continue
				}

				posInfo, ok := posResp.Data.(*bus.B_LP_V4_GetNftPosition_Response)
				if !ok {
					log.Error().Msg("get-nft-position: invalid response data")
					continue
				}

				// Skip zero liquidity positions
				if posInfo.Liquidity == nil || posInfo.Liquidity.Cmp(big.NewInt(0)) == 0 {
					continue
				}

				// Get token info from the on-chain poolKeys response
				currency0 := posInfo.Currency0
				currency1 := posInfo.Currency1
				fee := posInfo.Fee
				hookAddress := posInfo.HookAddress

				ui.Printf("%3d NFT Token: %v\n", found+1, tokenId)

				t0 := w.GetTokenByAddress(b.ChainId, currency0)
				if t0 != nil {
					ui.Printf("    %s (%s)", t0.Symbol, t0.Name)
				} else {
					ui.Printf("    ")
					cmn.AddAddressShortLink(ui.Terminal.Screen, currency0)
					ui.Printf(" ")
					ui.Terminal.Screen.AddLink(cmn.ICON_ADD, "command token add "+b.Name+" "+currency0.String(), "Add token", "")
				}

				ui.Printf(" / ")

				t1 := w.GetTokenByAddress(b.ChainId, currency1)
				if t1 != nil {
					ui.Printf("%s (%s)", t1.Symbol, t1.Name)
				} else {
					cmn.AddAddressShortLink(ui.Terminal.Screen, currency1)
					ui.Printf(" ")
					ui.Terminal.Screen.AddLink(cmn.ICON_ADD, "command token add "+b.Name+" "+currency1.String(), "Add token", "")
				}

				ui.Printf(" Fee: %.4f%%\n", float64(fee)/10000.)
				ui.Printf("    Ticks: %d / %d\n", posInfo.TickLower, posInfo.TickUpper)
				ui.Printf("    Liquidity: %s\n", posInfo.Liquidity.String())

				if hookAddress != (common.Address{}) {
					ui.Printf("    Hook: ")
					cmn.AddAddressShortLink(ui.Terminal.Screen, hookAddress)
					ui.Printf("\n")
				}

				ui.Flush()

				err = w.AddLP_V4Position(&cmn.LP_V4_Position{
					Owner:       addr.Address,
					ChainId:     pl.ChainId,
					Provider:    pl.Provider,
					PoolManager: pl.PoolManager,
					NFT_Token:   tokenId,
					PoolId:      posInfo.PoolId,
					Currency0:   currency0,
					Currency1:   currency1,
					Fee:         fee,
					TickSpacing: posInfo.TickSpacing,
					TickLower:   posInfo.TickLower,
					TickUpper:   posInfo.TickUpper,
					Liquidity:   posInfo.Liquidity,
					HookAddress: hookAddress,
				})

				if err != nil {
					log.Error().Err(err).Msg("AddLP_V4Position")
					return err
				}

				found++
			}
		}
	}

	ui.Printf("\nFound %d LP v4 positions\n", found)

	return nil
}

func querySubgraph(subgraphURL string, owner common.Address) ([]SubgraphPosition, error) {
	log.Debug().Msgf("querySubgraph URL: %s", subgraphURL)

	// Substitute API key placeholder if present
	if strings.Contains(subgraphURL, "{api-key}") {
		if cmn.Config.TheGraphAPIKey == "" {
			return nil, fmt.Errorf("subgraph URL contains {api-key} placeholder but TheGraphAPIKey is not configured. Set it via 'config set thegraph_api_key <your-key>'")
		}
		subgraphURL = strings.Replace(subgraphURL, "{api-key}", cmn.Config.TheGraphAPIKey, 1)
		log.Debug().Msgf("Substituted API key in URL: %s", subgraphURL)
	}

	query := fmt.Sprintf(`{
		positions(where: {owner: "%s"}, first: 1000) {
			id
			tokenId
			owner
			origin
			createdAtTimestamp
		}
	}`, strings.ToLower(owner.Hex()))

	log.Debug().Msgf("GraphQL query: %s", query)

	requestBody, err := json.Marshal(map[string]string{
		"query": query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	resp, err := http.Post(subgraphURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("subgraph request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	log.Debug().Msgf("Subgraph response status: %d, body: %s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subgraph returned status %d: %s", resp.StatusCode, string(body))
	}

	var subgraphResp SubgraphResponse
	if err := json.Unmarshal(body, &subgraphResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(subgraphResp.Errors) > 0 {
		return nil, fmt.Errorf("subgraph error: %s", subgraphResp.Errors[0].Message)
	}

	log.Debug().Msgf("Parsed %d positions from response", len(subgraphResp.Data.Positions))
	return subgraphResp.Data.Positions, nil
}

func queryPoolInfo(subgraphURL string, poolId string) (*SubgraphPool, error) {
	// Substitute API key placeholder if present
	if strings.Contains(subgraphURL, "{api-key}") {
		if cmn.Config.TheGraphAPIKey == "" {
			return nil, fmt.Errorf("subgraph URL contains {api-key} placeholder but TheGraphAPIKey is not configured")
		}
		subgraphURL = strings.Replace(subgraphURL, "{api-key}", cmn.Config.TheGraphAPIKey, 1)
	}

	query := fmt.Sprintf(`{
		pool(id: "%s") {
			id
			token0 { id symbol }
			token1 { id symbol }
			feeTier
			hooks
		}
	}`, strings.ToLower(poolId))

	requestBody, err := json.Marshal(map[string]string{
		"query": query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	resp, err := http.Post(subgraphURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("subgraph request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subgraph returned status %d: %s", resp.StatusCode, string(body))
	}

	var poolResp SubgraphPoolResponse
	if err := json.Unmarshal(body, &poolResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(poolResp.Errors) > 0 {
		return nil, fmt.Errorf("subgraph error: %s", poolResp.Errors[0].Message)
	}

	return poolResp.Data.Pool, nil
}
