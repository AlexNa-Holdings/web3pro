package command

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

var lp_v3_subcommands = []string{
	"on", "off", "add", "remove", "discover", "providers",
	"list",
}

func NewLP_V3Command() *Command {
	return &Command{
		Command:      "lp_v3",
		ShortCommand: "v3",
		Usage: `
Usage: liquidity v3 [COMMAND]

Manage v3 liquidity 

Commands:
  list                      - List v3 positions
  providers				    - List v3 providers
  add [CHAIN] [ADDR] [NAME] - Add v3 provider
  remove [CHAIN] [ADDR]     - Remove v3 provider
  discover [CHAIN] [Name]   - Discover v3 positions
  on                        - Open v3 window
  off                       - Close w3 window
		`,
		Help:             `Manage liquidity v3`,
		Process:          LP_V3_Process,
		AutoCompleteFunc: LP_V3_AutoComplete,
	}
}

func LP_V3_AutoComplete(input string) (string, *[]ui.ACOption, string) {

	if cmn.CurrentWallet == nil {
		return "", nil, ""
	}

	w := cmn.CurrentWallet

	options := []ui.ACOption{}
	p := cmn.SplitN(input, 5)
	command, subcommand, bchain, addr, _ := p[0], p[1], p[2], p[3], p[4]

	last_param := len(p) - 1
	for last_param > 0 && p[last_param] == "" {
		last_param--
	}

	if strings.HasSuffix(input, " ") {
		last_param++
	}

	switch last_param {
	case 1:
		if !cmn.IsInArray(lp_v3_subcommands, subcommand) {
			for _, sc := range lp_v3_subcommands {
				if input == "" || strings.Contains(sc, subcommand) {
					options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
				}
			}
			return "action", &options, subcommand
		}
	case 2:
		if subcommand == "add" || subcommand == "remove" || subcommand == "discover" {
			for _, chain := range w.Blockchains {
				if cmn.Contains(chain.Name, bchain) {
					options = append(options, ui.ACOption{
						Name:   chain.Name,
						Result: command + " " + subcommand + " '" + chain.Name + "' "})
				}
			}
			return "blockchain", &options, bchain

		}
	case 3:
		if subcommand == "add" {
			b := w.GetBlockchain(bchain)
			if b != nil {
				for _, lp := range cmn.PrefedinedLP_V3[b.ChainId] {
					options = append(options, ui.ACOption{
						Name:   lp.Name,
						Result: command + " " + subcommand + " '" + b.Name + "' '" + lp.Address.Hex() + "' '" + lp.Name + "' '" + lp.URL + "'"})
				}

				return "address", &options, addr
			}
		}

		if subcommand == "discover" {
			for _, lp := range w.LP_V3_Providers {
				if cmn.Contains(lp.Name, addr) {
					options = append(options, ui.ACOption{
						Name:   lp.Name,
						Result: command + " " + subcommand + " " + strconv.Itoa(lp.ChainId) + " '" + lp.Name + "'"})
				}
			}
			return "name", &options, addr
		}

		if subcommand == "remove" {
			b := w.GetBlockchain(bchain)
			if b != nil {
				for _, lp := range w.LP_V3_Providers {
					if lp.ChainId == b.ChainId && cmn.Contains(lp.Name, addr) {
						options = append(options, ui.ACOption{
							Name:   lp.Name,
							Result: command + " " + subcommand + " " + strconv.Itoa(lp.ChainId) + " '" + lp.Name + "'"})
					}
				}
			}
			return "address", &options, addr
		}

	}
	return "", &options, ""
}

func LP_V3_Process(c *Command, input string) {
	var err error
	if cmn.CurrentWallet == nil {
		ui.PrintErrorf("No wallet open\n")
		return
	}

	w := cmn.CurrentWallet
	if w == nil {
		ui.PrintErrorf("No wallet open\n")
		return
	}

	p := cmn.SplitN(input, 6)
	_, subcommand, chain, addr, name, url := p[0], p[1], p[2], p[3], p[4], p[5]

	switch subcommand {
	case "list", "":
		list(w)
	case "providers":
		ui.Printf("\nLP v3 Providers\n\n")

		if len(w.LP_V3_Providers) == 0 {
			ui.Printf("(no providers)\n")
		}

		sort.Slice(w.LP_V3_Providers, func(i, j int) bool {
			if w.LP_V3_Providers[i].ChainId == w.LP_V3_Providers[j].ChainId {
				return w.LP_V3_Providers[i].Name < w.LP_V3_Providers[j].Name
			}
			return w.LP_V3_Providers[i].ChainId < w.LP_V3_Providers[j].ChainId
		})

		for i, lp := range w.LP_V3_Providers {
			b := w.GetBlockchainById(lp.ChainId)
			if b == nil {
				ui.PrintErrorf("Blockchain not found: %d", lp.ChainId)
				w.RemoveLP_V3(lp.ChainId, lp.Provider)
				break
			}

			ui.Printf("%d %12s %s ", i+1, b.Name, lp.Name)
			ui.Terminal.Screen.AddLink(gocui.ICON_EDIT, "command lp_v3 edit "+strconv.Itoa(lp.ChainId)+" '"+lp.Provider.Hex()+"' '"+lp.Name+"'", "Edit provider", "")
			ui.Terminal.Screen.AddLink(gocui.ICON_DELETE, "command lp_v3 remove "+strconv.Itoa(lp.ChainId)+" '"+lp.Provider.Hex()+"'", "Remove provider", "")
			cmn.AddAddressShortLink(ui.Terminal.Screen, lp.Provider)
		}

		ui.Printf("\n")

	case "add":
		b := w.GetBlockchain(chain)
		if b == nil {
			err = fmt.Errorf("blockchain not found: %s", chain)
			break
		}

		bus.Send("ui", "popup", ui.DlgLP_V3_add(b, addr, name, url))
	case "edit":
		b := w.GetBlockchain(chain)
		if b == nil {
			err = fmt.Errorf("blockchain not found: %s", chain)
			break
		}

		bus.Send("ui", "popup", ui.DlgLP_V3_edit(b, addr, name, url))
	case "remove":
		b := w.GetBlockchain(chain)
		if b == nil {
			err = fmt.Errorf("blockchain not found: %s", chain)
			break
		}

		lp := w.GetLP_V3_by_name(b.ChainId, addr)
		if lp == nil {
			lp := w.GetLP_V3(b.ChainId, common.HexToAddress(addr))
			if lp == nil {
				err = fmt.Errorf("provider not found: %s", addr)
				break
			}
		}

		bus.Send("ui", "popup", ui.DlgConfirm(
			"Remove provider",
			`	
<c>Are you sure you want to remove provider?</c>

       Name:`+lp.Name+`
 Blockchain:`+b.Name+`
    Address:`+lp.Provider.String()+`
`,
			func() {
				err := w.RemoveLP_V3(b.ChainId, lp.Provider)
				if err != nil {
					ui.PrintErrorf("Error removing provider: %v", err)
					return
				}
				ui.Notification.Show("Provider removed")
			}))

	case "discover":
		chain_id := 0
		b := w.GetBlockchain(chain)
		if b != nil {
			chain_id = b.ChainId
		}

		resp := bus.Fetch("lp_v3", "discover", bus.B_LP_V3_Discover{
			ChainId: chain_id,
			Name:    name,
		})
		if resp.Error != nil {
			err = resp.Error
		}

	default:
		err = fmt.Errorf("unknown command: %s", subcommand)
	}

	if err != nil {
		ui.PrintErrorf(err.Error())
	}

}

func list(w *cmn.Wallet) {
	ui.Printf("\nLP v3 Positions\n\n")

	if len(w.LP_V3_Positions) == 0 {
		ui.Printf("(no positions)\n")
	}

	sort.Slice(w.LP_V3_Positions, func(i, j int) bool {
		if w.LP_V3_Positions[i].ChainId == w.LP_V3_Positions[j].ChainId {
			p1 := w.GetLP_V3(w.LP_V3_Positions[i].ChainId, w.LP_V3_Positions[i].Provider)
			p2 := w.GetLP_V3(w.LP_V3_Positions[j].ChainId, w.LP_V3_Positions[j].Provider)
			if p1 != nil && p2 != nil {
				return p1.Name < p2.Name
			} else {
				return w.LP_V3_Positions[i].Provider.Hex() < w.LP_V3_Positions[j].Provider.Hex()
			}

		}
		return w.LP_V3_Positions[i].ChainId < w.LP_V3_Positions[j].ChainId
	})

	ui.Printf("Contract|Chain    Pair    On Liq0     Liq1     Gain0    Gain1     Gain$    Address\n")

	for _, lp := range w.LP_V3_Positions {

		// sanity check
		if lp.Address.Cmp(common.Address{}) == 0 {
			ui.PrintErrorf("No address, LP v3 position removed")
			w.RemoveLP_V3Position(lp.Address, lp.ChainId, lp.Provider, lp.NFT_Token)
			log.Error().Msgf("No address, LP v3 position removed")
			continue
		}

		b := w.GetBlockchainById(lp.ChainId)
		if b == nil {
			ui.PrintErrorf("Blockchain not found, V3 position removed: %d", lp.ChainId)
			w.RemoveLP_V3Position(lp.Address, lp.ChainId, lp.Provider, lp.NFT_Token)
			log.Error().Msgf("Blockchain not found, V3 position removed: %d", lp.ChainId)
			continue
		}

		lpp := w.GetLP_V3(lp.ChainId, lp.Provider)
		if lpp == nil {
			ui.PrintErrorf("Provider not found, V3 position removed: %s", lp.Provider.String())
			w.RemoveLP_V3Position(lp.Address, lp.ChainId, lp.Provider, lp.NFT_Token)
			log.Error().Msgf("Provider not found, V3 position removed: %s", lp.Provider.String())
			continue
		}

		a := w.GetAddress(lp.Address)
		if a == nil {
			ui.PrintErrorf("Address not found, V3 position removed: %s", lp.Address.String())
			w.RemoveLP_V3Position(lp.Address, lp.ChainId, lp.Provider, lp.NFT_Token)
			log.Error().Msgf("Address not found, V3 position removed: %s", lp.Address.String())
			continue
		}

		// Fetch position
		pos_resp := bus.Fetch("lp_v3", "get-position", &bus.B_LP_V3_GetPosition{
			ChainId:   lp.ChainId,
			Provider:  lp.Provider,
			From:      w.CurrentAddress,
			NFT_Token: lp.NFT_Token,
		})

		if pos_resp.Error != nil {
			ui.PrintErrorf("Error fetching position: %v", pos_resp.Error)
			continue
		}

		pos, ok := pos_resp.Data.(*bus.B_LP_V3_GetPosition_Response)
		if !ok {
			ui.PrintErrorf("Invalid data")
			continue
		}

		// Fetch price
		price_resp := bus.Fetch("lp_v3", "get-price", &bus.B_LP_V3_GetPrice{
			ChainId: lp.ChainId,
			Pool:    lp.Pool,
		})

		if price_resp.Error != nil {
			ui.PrintErrorf("Error fetching price: %v", price_resp.Error)
			continue
		}

		price, ok := price_resp.Data.(*big.Int)
		if !ok {
			ui.PrintErrorf("Invalid data")
			continue
		}

		amount0, amount1, in_range := calculateAmounts(pos.Liquidity, price,
			getSqrtPriceX96FromTick(pos.TickLower.Int64()),
			getSqrtPriceX96FromTick(pos.TickUpper.Int64()))

		ui.Printf("%-16s ", lpp.Name+"|"+b.Currency)

		t0 := w.GetTokenByAddress(b.Name, lp.Token0)
		t1 := w.GetTokenByAddress(b.Name, lp.Token1)

		if t0 != nil && t1 != nil {
			ui.Printf("%9s", t0.Symbol+"\uf0ec "+t1.Symbol)
		} else {
			if t0 != nil {
				ui.Printf("%-5s", t0.Symbol)
			} else {
				cmn.AddAddressShortLink(ui.Terminal.Screen, lp.Token0)
				ui.Terminal.Screen.AddLink(gocui.ICON_ADD, "command token add "+b.Name+" "+lp.Token0.String(), "Add token", "")
			}

			ui.Printf("\uf0ec ")

			if t1 != nil {
				ui.Printf("%-5s", t1.Symbol)
			} else {
				cmn.AddAddressShortLink(ui.Terminal.Screen, lp.Token1)
				ui.Terminal.Screen.AddLink(gocui.ICON_ADD, "command token add "+b.Name+" "+lp.Token1.String(), "Add token", "")
			}
			ui.Printf(" %s\n", a.Name)
			continue
		}

		if in_range {
			ui.Printf(ui.F(gocui.ColorGreen) + gocui.ICON_LIGHT + ui.F(ui.Terminal.Screen.FgColor))
		} else {
			ui.Printf(ui.F(gocui.ColorRed) + gocui.ICON_LIGHT + ui.F(ui.Terminal.Screen.FgColor))
		}

		cmn.AddValueLink(ui.Terminal.Screen, amount0, t0)
		cmn.AddValueLink(ui.Terminal.Screen, amount1, t1)

		cmn.AddValueLink(ui.Terminal.Screen, pos.TokensOwed0, t0)
		cmn.AddValueLink(ui.Terminal.Screen, pos.TokensOwed1, t1)

		dollars := t0.Float64(pos.TokensOwed0)*t0.Price +
			t1.Float64(pos.TokensOwed1)*t1.Price

		cmn.AddDollarLink(ui.Terminal.Screen, dollars)

		// cmn.AddAddressShortLink(ui.Terminal.Screen, a.Address)
		ui.Printf(" %s\n", a.Name)

	}

	ui.Printf("\n")
}

func getSqrtPriceX96FromTick(tick int64) *big.Int {
	// Calculate 1.0001^tick as a float
	price := math.Pow(1.0001, float64(tick))

	// Take the square root of the price
	sqrtPrice := math.Sqrt(price)

	// Multiply by 2^96 to convert to Q96 format
	two96 := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(2), big.NewInt(96), nil))
	sqrtPriceX96Float := new(big.Float).Mul(big.NewFloat(sqrtPrice), two96)

	// Convert to *big.Int
	sqrtPriceX96 := new(big.Int)
	sqrtPriceX96Float.Int(sqrtPriceX96)

	return sqrtPriceX96
}

func calculateAmounts(liquidity, sqrtPriceX96, tickLowerSqrtPriceX96, tickUpperSqrtPriceX96 *big.Int) (*big.Int, *big.Int, bool) {
	in_range := false
	two96 := new(big.Int).Exp(big.NewInt(2), big.NewInt(96), nil)

	amount0 := big.NewInt(0)
	amount1 := big.NewInt(0)

	// Check if sqrtPriceX96 is within tickLower and tickUpper
	if sqrtPriceX96.Cmp(tickLowerSqrtPriceX96) <= 0 {
		// Price is below the range: Only token0 is involved
		numerator := new(big.Int).Sub(tickUpperSqrtPriceX96, tickLowerSqrtPriceX96)
		numerator.Mul(numerator, liquidity)

		denominator := new(big.Int).Mul(tickLowerSqrtPriceX96, tickUpperSqrtPriceX96)
		amount0.Div(numerator, denominator)
	} else if sqrtPriceX96.Cmp(tickUpperSqrtPriceX96) >= 0 {
		// Price is above the range: Only token1 is involved
		numerator := new(big.Int).Sub(tickUpperSqrtPriceX96, tickLowerSqrtPriceX96)
		numerator.Mul(numerator, liquidity)

		amount1.Mul(numerator, two96)
	} else {
		in_range = true
		// Price is within the range: Both tokens are involved

		// Calculate amount0
		amount0Numerator := new(big.Int).Sub(tickUpperSqrtPriceX96, sqrtPriceX96)
		amount0Numerator.Mul(amount0Numerator, liquidity)

		// Keep precision high by multiplying first and dividing last
		denominator0 := new(big.Int).Mul(sqrtPriceX96, tickUpperSqrtPriceX96)

		// Ensure numerator is multiplied by `2^96` to match precision
		amount0Numerator.Mul(amount0Numerator, two96)
		amount0.Div(amount0Numerator, denominator0)

		// Calculate amount1
		amount1Numerator := new(big.Int).Sub(sqrtPriceX96, tickLowerSqrtPriceX96)
		amount1Numerator.Mul(amount1Numerator, liquidity)

		amount1.Div(amount1Numerator, two96)
	}

	return amount0, amount1, in_range
}
