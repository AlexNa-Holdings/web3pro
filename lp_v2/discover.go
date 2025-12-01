package lp_v2

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

// GraphQL response structures for V2 subgraph
type V2SubgraphResponse struct {
	Data   V2SubgraphData    `json:"data"`
	Errors []V2SubgraphError `json:"errors,omitempty"`
}

type V2SubgraphError struct {
	Message string `json:"message"`
}

type V2SubgraphData struct {
	LiquidityPositions []V2LiquidityPosition `json:"liquidityPositions"`
}

type V2LiquidityPosition struct {
	ID                  string  `json:"id"`
	LiquidityTokenBalance string `json:"liquidityTokenBalance"`
	Pair                V2Pair  `json:"pair"`
}

type V2Pair struct {
	ID      string   `json:"id"`
	Token0  V2Token  `json:"token0"`
	Token1  V2Token  `json:"token1"`
}

type V2Token struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
}

func discover(msg *bus.Message) error {
	found := 0

	req, ok := msg.Data.(bus.B_LP_V2_Discover)
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

	ui.Printf("Discovering LP v2\n\n")

	log.Debug().Msgf("LP_V2 Providers count: %d", len(w.LP_V2_Providers))
	if len(w.LP_V2_Providers) == 0 {
		ui.Printf("No LP v2 providers configured. Use 'lp_v2 add' to add a provider first.\n")
		return nil
	}

	for _, pl := range w.LP_V2_Providers {
		log.Debug().Msgf("Checking provider: %s, ChainId: %d, SubgraphID: %s", pl.Name, pl.ChainId, pl.SubgraphID)
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

		if pl.SubgraphID == "" {
			ui.Printf("Skipping %s %s: no subgraph ID configured\n", b.Name, pl.Name)
			continue
		}

		// Build full subgraph URL from gateway + ID
		subgraphURL := cmn.Config.TheGraphGateway + pl.SubgraphID

		ui.Printf("Discovering LP v2: %s %s\n", b.Name, pl.Name)

		for _, addr := range w.Addresses {
			ui.Printf("  ")
			cmn.AddAddressShortLink(ui.Terminal.Screen, addr.Address)
			ui.Printf(" %s \n", addr.Name)
			ui.Flush()

			log.Debug().Msgf("Querying subgraph for address: %s", addr.Address.Hex())
			positions, err := queryV2Subgraph(subgraphURL, addr.Address)
			if err != nil {
				log.Error().Err(err).Msg("queryV2Subgraph")
				ui.PrintErrorf("Subgraph query failed: %v\n", err)
				continue
			}
			log.Debug().Msgf("Found %d positions for address %s", len(positions), addr.Address.Hex())

			for _, pos := range positions {
				// Parse liquidity balance
				balance, ok := new(big.Float).SetString(pos.LiquidityTokenBalance)
				if !ok || balance.Cmp(big.NewFloat(0)) == 0 {
					continue // Skip zero balance positions
				}

				pairAddr := common.HexToAddress(pos.Pair.ID)
				token0Addr := common.HexToAddress(pos.Pair.Token0.ID)
				token1Addr := common.HexToAddress(pos.Pair.Token1.ID)

				ui.Printf("%3d Pair: ", found+1)
				cmn.AddAddressShortLink(ui.Terminal.Screen, pairAddr)
				ui.Printf("\n")

				t0 := w.GetTokenByAddress(b.ChainId, token0Addr)
				if t0 != nil {
					ui.Printf("    %s (%s)", t0.Symbol, t0.Name)
				} else {
					ui.Printf("    ")
					cmn.AddAddressShortLink(ui.Terminal.Screen, token0Addr)
					ui.Printf(" ")
					ui.Terminal.Screen.AddLink(cmn.ICON_ADD, "command token add "+b.Name+" "+token0Addr.String(), "Add token", "")
				}

				ui.Printf(" / ")

				t1 := w.GetTokenByAddress(b.ChainId, token1Addr)
				if t1 != nil {
					ui.Printf("%s (%s)", t1.Symbol, t1.Name)
				} else {
					cmn.AddAddressShortLink(ui.Terminal.Screen, token1Addr)
					ui.Printf(" ")
					ui.Terminal.Screen.AddLink(cmn.ICON_ADD, "command token add "+b.Name+" "+token1Addr.String(), "Add token", "")
				}

				ui.Printf("\n    LP Balance: %s\n", pos.LiquidityTokenBalance)

				ui.Flush()

				err = w.AddLP_V2Position(&cmn.LP_V2_Position{
					Owner:   addr.Address,
					ChainId: pl.ChainId,
					Factory: pl.Factory,
					Pair:    pairAddr,
					Token0:  token0Addr,
					Token1:  token1Addr,
				})

				if err != nil {
					log.Error().Err(err).Msg("AddLP_V2Position")
					return err
				}

				found++
			}
		}
	}

	ui.Printf("\nFound %d LP v2 positions\n", found)

	return nil
}

func queryV2Subgraph(subgraphURL string, owner common.Address) ([]V2LiquidityPosition, error) {
	log.Debug().Msgf("queryV2Subgraph URL: %s", subgraphURL)

	// Substitute API key placeholder if present
	if strings.Contains(subgraphURL, "{api-key}") {
		if cmn.Config.TheGraphAPIKey == "" {
			return nil, fmt.Errorf("subgraph URL contains {api-key} placeholder but TheGraphAPIKey is not configured. Set it via 'config set thegraph_api_key <your-key>'")
		}
		subgraphURL = strings.Replace(subgraphURL, "{api-key}", cmn.Config.TheGraphAPIKey, 1)
		log.Debug().Msgf("Substituted API key in URL: %s", subgraphURL)
	}

	query := fmt.Sprintf(`{
		liquidityPositions(where: {user: "%s", liquidityTokenBalance_gt: "0"}, first: 1000) {
			id
			liquidityTokenBalance
			pair {
				id
				token0 { id symbol }
				token1 { id symbol }
			}
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

	var subgraphResp V2SubgraphResponse
	if err := json.Unmarshal(body, &subgraphResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(subgraphResp.Errors) > 0 {
		return nil, fmt.Errorf("subgraph error: %s", subgraphResp.Errors[0].Message)
	}

	log.Debug().Msgf("Parsed %d positions from response", len(subgraphResp.Data.LiquidityPositions))
	return subgraphResp.Data.LiquidityPositions, nil
}
