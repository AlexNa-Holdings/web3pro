package lp_v3

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

func discover(msg *bus.Message) error {
	found := 0

	req, ok := msg.Data.(bus.B_LP_V3_Discover)
	if !ok {
		return fmt.Errorf("invalid tx: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return fmt.Errorf("discover: no wallet")
	}

	chain_id := 0

	b := w.GetBlockchainById(req.ChainId)
	if b != nil {
		chain_id = b.ChainId
	}

	name := req.Name

	ui.Printf("Discovering LP v3\n\n")

	for _, pl := range w.LP_V3_Providers {

		if name != "" && pl.Name != name {
			continue
		}

		if chain_id != 0 && pl.ChainId != chain_id {
			continue
		}

		b := w.GetBlockchainById(pl.ChainId)
		if b == nil {
			log.Error().Msgf("Blockchain not found: %d", pl.ChainId)
			continue
		}

		ui.Printf("Discovering LP v3: %s %s\n", b.Name, pl.Name)

		for _, addr := range w.Addresses {

			ui.Printf("  ")
			cmn.AddAddressShortLink(ui.Terminal.Screen, addr.Address)
			ui.Printf(" %s \n", addr.Name)
			ui.Flush()

			data, err := V3_MANAGER.Pack("balanceOf", addr.Address)
			if err != nil {
				log.Error().Err(err).Msg("V3_ABI.Pack balanceOf")
				return err
			}

			resp := bus.Fetch("eth", "call", &bus.B_EthCall{
				ChainId: pl.ChainId,
				To:      pl.Provider,
				From:    addr.Address,
				Data:    data,
			})

			if resp.Error != nil {
				log.Error().Err(resp.Error).Msg("eth call")
				return resp.Error
			}

			n := cmn.Uint256FromHex(resp.Data.(string))

			for i := 0; i < int(n.Uint64()); i++ {
				data, err := V3_MANAGER.Pack("tokenOfOwnerByIndex", addr.Address, big.NewInt(int64(i)))
				if err != nil {
					log.Error().Err(err).Msg("V3_ABI.Pack tokenByIndex")
					return err
				}

				resp := bus.Fetch("eth", "call", &bus.B_EthCall{
					ChainId: pl.ChainId,
					To:      pl.Provider,
					From:    addr.Address,
					Data:    data,
				})

				if resp.Error != nil {
					log.Error().Err(resp.Error).Msg("eth call")
					return resp.Error
				}

				token := cmn.Uint256FromHex(resp.Data.(string))

				pos_resp := msg.Fetch("lp_v3", "get-nft-position", &bus.B_LP_V3_GetNftPosition{
					ChainId:   pl.ChainId,
					Provider:  pl.Provider,
					From:      addr.Address,
					NFT_Token: token,
				})

				if pos_resp.Error != nil {
					log.Error().Err(pos_resp.Error).Msg("get_position")
					return pos_resp.Error
				}

				pos, ok := pos_resp.Data.(*bus.B_LP_V3_GetNftPosition_Response)
				if !ok {
					log.Error().Msg("get_position: invalid data")
					return fmt.Errorf("get_position: invalid data")
				}

				if pos.Liquidity.Cmp(big.NewInt(0)) == 0 && pos.TokensOwed0.Cmp(big.NewInt(0)) == 0 && pos.TokensOwed1.Cmp(big.NewInt(0)) == 0 {
					// no liquidity no gains
					continue
				}

				ui.Printf("%3d NFT Token: %v Operator: ", found+1, token)
				cmn.AddAddressShortLink(ui.Terminal.Screen, pos.Operator)
				ui.Printf("\n")

				t0 := w.GetTokenByAddress(b.ChainId, pos.Token0)
				if t0 != nil {
					ui.Printf("    %s (%s)", t0.Symbol, t0.Name)
				} else {
					ui.Printf("    ")
					cmn.AddAddressShortLink(ui.Terminal.Screen, pos.Token0)
					ui.Printf(" ")
					ui.Terminal.Screen.AddLink(gocui.ICON_ADD, "command token add "+b.Name+" "+pos.Token0.String(), "Add token", "")
				}

				ui.Printf(" / ")

				t1 := w.GetTokenByAddress(b.ChainId, pos.Token1)
				if t1 != nil {
					ui.Printf("%s (%s)", t1.Symbol, t1.Name)
				} else {
					cmn.AddAddressShortLink(ui.Terminal.Screen, pos.Token1)
					ui.Printf(" ")
					ui.Terminal.Screen.AddLink(gocui.ICON_ADD, "command token add "+b.Name+" "+pos.Token1.String(), "Add token", "")
				}

				ui.Printf(" Fee: %.2f%%\n", float64(pos.Fee.Uint64())/10000.)
				ui.Printf("    Price Ticks: %d / %d\n", pos.TickLower, pos.TickUpper)
				ui.Printf("    Liquidity: %s\n", pos.Liquidity.String())
				// ui.Printf("    FeeGrowthInside0LastX128: %s\n", feeGrowthInside0LastX128.String())
				// ui.Printf("    FeeGrowthInside1LastX128: %s\n", feeGrowthInside1LastX128.String())
				// ui.Printf("    Owed: ")

				// if t0 != nil {
				// 	cmn.AddValueSymbolLink(ui.Terminal.Screen, pos.TokensOwed0, t0)
				// } else {
				// 	ui.Printf("%s", pos.TokensOwed0.String())
				// }

				// ui.Printf(" / ")

				// if t1 != nil {
				// 	cmn.AddValueSymbolLink(ui.Terminal.Screen, pos.TokensOwed1, t1)
				// } else {
				// 	ui.Printf("%s", pos.TokensOwed1.String())
				// }

				// get factory
				factory_resp := msg.Fetch("lp_v3", "get-factory", &bus.B_LP_V3_GetFactory{
					ChainId:  pl.ChainId,
					Provider: pl.Provider,
				})

				if factory_resp.Error != nil {
					log.Error().Err(factory_resp.Error).Msg("get_factory")
					return factory_resp.Error
				}

				factory, ok := factory_resp.Data.(common.Address)
				if !ok {
					log.Error().Msg("get_factory: invalid data")
					return fmt.Errorf("get_factory: invalid data")
				}

				// get pool
				pool_resp := msg.Fetch("lp_v3", "get-pool", &bus.B_LP_V3_GetPool{
					ChainId: pl.ChainId,
					Factory: factory,
					Token0:  pos.Token0,
					Token1:  pos.Token1,
					Fee:     pos.Fee,
				})

				if pool_resp.Error != nil {
					log.Error().Err(pool_resp.Error).Msg("get_pool")
					return pool_resp.Error
				}

				pool, ok := pool_resp.Data.(common.Address)
				if !ok {
					log.Error().Msg("get_pool: invalid data")
					return fmt.Errorf("get_pool: invalid data")
				}

				ui.Printf("    Pool: ")
				cmn.AddAddressShortLink(ui.Terminal.Screen, pool)

				ui.Printf("\n")

				ui.Flush()

				err = w.AddLP_V3Position(&cmn.LP_V3_Position{
					Owner:     addr.Address,
					ChainId:   pl.ChainId,
					Provider:  pl.Provider,
					NFT_Token: token,
					Token0:    pos.Token0,
					Token1:    pos.Token1,
					Fee:       pos.Fee,
					Pool:      pool,
				})

				if err != nil {
					log.Error().Err(err).Msg("AddLP_V3_Position")
					return err
				}

				found++
			}

		}
	}

	ui.Printf("\nFound %d LP v3 positions\n", found)

	return nil
}
