package lp_v3

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

func discover(msg *bus.Message) error {

	req, ok := msg.Data.(bus.B_LP_V3_Discover)
	if !ok {
		return fmt.Errorf("invalid tx: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return fmt.Errorf("discover: no wallet")
	}

	chain_id := -1

	b := w.GetBlockchain(req.Chain)
	if b != nil {
		chain_id = b.ChainID
	}

	name := req.Name

	ui.Printf("Discovering LP v3\n\n")

	for _, pl := range w.LP_V3_Providers {

		b := w.GetBlockchain(pl.Blockchain)
		if b == nil {
			return fmt.Errorf("discover: blockchain not found: %v", pl.Blockchain)
		}

		if chain_id != -1 && b.ChainID != chain_id {
			continue
		}

		if name != "" && pl.Name != name {
			continue
		}

		ui.Printf("Discovering LP v3: %s %s\n", pl.Blockchain, pl.Name)

		found := 0

		for _, addr := range w.Addresses {

			ui.Printf("  ")
			cmn.AddAddressShortLink(ui.Terminal.Screen, addr.Address)
			ui.Printf(" %s \n", addr.Name)
			ui.Flush()

			data, err := V3_ABI.Pack("balanceOf", addr.Address)
			if err != nil {
				log.Error().Err(err).Msg("V3_ABI.Pack balanceOf")
				return err
			}

			resp := bus.Fetch("eth", "call", &bus.B_EthCall{
				Blockchain: pl.Blockchain,
				To:         pl.Address,
				From:       addr.Address,
				Data:       data,
			})

			if resp.Error != nil {
				log.Error().Err(resp.Error).Msg("eth call")
				return resp.Error
			}

			n := cmn.Uint256FromHex(resp.Data.(string))

			for i := 0; i < int(n.Uint64()); i++ {
				data, err := V3_ABI.Pack("tokenOfOwnerByIndex", addr.Address, big.NewInt(int64(i)))
				if err != nil {
					log.Error().Err(err).Msg("V3_ABI.Pack tokenByIndex")
					return err
				}

				resp := bus.Fetch("eth", "call", &bus.B_EthCall{
					Blockchain: pl.Blockchain,
					To:         pl.Address,
					From:       addr.Address,
					Data:       data,
				})

				if resp.Error != nil {
					log.Error().Err(resp.Error).Msg("eth call")
					return resp.Error
				}

				token := cmn.Uint256FromHex(resp.Data.(string))

				data, err = V3_ABI.Pack("positions", token)
				if err != nil {
					log.Error().Err(err).Msg("V3_ABI.Pack positions")
					return err
				}

				resp = bus.Fetch("eth", "call", &bus.B_EthCall{
					Blockchain: pl.Blockchain,
					To:         pl.Address,
					From:       addr.Address,
					Data:       data,
				})

				if resp.Error != nil {
					log.Error().Err(resp.Error).Msg("eth call")
					return resp.Error
				}

				var (
					nonce                                              *big.Int
					operator                                           common.Address
					token0                                             common.Address
					token1                                             common.Address
					fee                                                *big.Int
					tickLower, tickUpper                               *big.Int
					liquidity                                          *big.Int
					feeGrowthInside0LastX128, feeGrowthInside1LastX128 *big.Int
					tokensOwed0, tokensOwed1                           *big.Int
				)

				output, err := hexutil.Decode(resp.Data.(string))
				if err != nil {
					log.Error().Err(err).Msg("hexutil.Decode")
					return err
				}

				err = V3_ABI.UnpackIntoInterface(
					&[]interface{}{
						&nonce,
						&operator,
						&token0,
						&token1,
						&fee,
						&tickLower,
						&tickUpper,
						&liquidity,
						&feeGrowthInside0LastX128,
						&feeGrowthInside1LastX128,
						&tokensOwed0,
						&tokensOwed1,
					}, "positions", output)

				if err != nil {
					log.Error().Err(err).Msg("positionManagerABI.UnpackIntoInterface")
					return err
				}

				if liquidity.Cmp(big.NewInt(0)) == 0 && tokensOwed0.Cmp(big.NewInt(0)) == 0 && tokensOwed1.Cmp(big.NewInt(0)) == 0 {
					// no liquidity no gains
					continue
				}

				ui.Printf("    NFT Token: %v\n", token)
				// ui.Printf("    Operator: %s\n", operator.String())

				t0 := w.GetTokenByAddress(b.Name, token0)
				if t0 != nil {
					ui.Printf("    Token0: %s (%s)\n", t0.Symbol, t0.Name)
				} else {
					ui.Printf("    Token0: %s ", token0.String())
					ui.Terminal.Screen.AddLink(gocui.ICON_ADD, "command token add "+b.Name+" "+token0.String(), "Add token", "")
					ui.Printf("\n")
				}

				t1 := w.GetTokenByAddress(b.Name, token1)
				if t1 != nil {
					ui.Printf("    Token1: %s (%s)\n", t1.Symbol, t1.Name)
				} else {
					ui.Printf("    Token1: %s ", token1.String())
					ui.Terminal.Screen.AddLink(gocui.ICON_ADD, "command token add "+b.Name+" "+token1.String(), "Add token", "")
					ui.Printf("\n")
				}

				ui.Printf("    Fee: %.2f%%\n", float64(fee.Uint64())/10000.)
				ui.Printf("    Price Ticks: %s - %s\n", tickLower.String(), tickUpper.String())
				ui.Printf("    Liquidity: %s\n", liquidity.String())
				// ui.Printf("    FeeGrowthInside0LastX128: %s\n", feeGrowthInside0LastX128.String())
				// ui.Printf("    FeeGrowthInside1LastX128: %s\n", feeGrowthInside1LastX128.String())
				ui.Printf("    TokensOwed0: %s\n", tokensOwed0.String())
				ui.Printf("    TokensOwed1: %s\n", tokensOwed1.String())

				ui.Printf("\n")

				ui.Flush()
				found++
			}

		}

		ui.Printf("\nFound %d LP v3 positions\n", found)

	}

	return nil
}
