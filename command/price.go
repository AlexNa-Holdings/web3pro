package command

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/price"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/rs/zerolog/log"
)

var price_subcommands = []string{"set_feeder", "discover", "update", "list"}

func NewPriceCommand() *Command {
	return &Command{
		Command:      "price",
		ShortCommand: "p",
		Usage: `
Usage: 

  discover [BLOCKCHAIN] [TOKEN_ADDR]                   - Discover trading pairs for token
  set_feeder [BLOCKCHAIN] [PAIR_ADDR] [FEEDER] [PARAM} - Set price feeder for trading pair
  update                                               - Update price feeders
  list                                                 - List price feeders

		`,
		Help:             `Price Feeder`,
		Process:          Price_Process,
		AutoCompleteFunc: Price_AutoComplete,
	}
}

func Price_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	if cmn.CurrentWallet == nil {
		return "", nil, ""
	}

	w := cmn.CurrentWallet

	options := []ui.ACOption{}
	p := cmn.SplitN(input, 6)
	command, subcommand, bchain, token := p[0], p[1], p[2], p[3]

	last_param := len(p) - 1
	for last_param > 0 && p[last_param] == "" {
		last_param--
	}

	if strings.HasSuffix(input, " ") {
		last_param++
	}

	switch last_param {
	case 0, 1:

		if !cmn.IsInArray(price_subcommands, subcommand) {
			for _, sc := range price_subcommands {
				if input == "" || strings.Contains(sc, subcommand) {
					options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
				}
			}
			return "action", &options, subcommand
		}
	case 2:

		if subcommand == "discover" || subcommand == "set_feeder" {

			b := w.GetBlockchainByName(bchain)

			if token == "" && b == nil {
				for _, b := range w.Blockchains {
					if cmn.Contains(b.Name, bchain) {
						options = append(options,
							ui.ACOption{Name: b.Name, Result: command + " " + subcommand + " " + strconv.Itoa(b.ChainId) + " "})
					}
				}
				return "blockchain", &options, bchain
			}
		}

	case 3:
		if subcommand == "discover" || subcommand == "set_feeder" {

			b := w.GetBlockchainByName(bchain)

			if b != nil {
				for _, t := range w.Tokens {
					if t.ChainId == b.ChainId && cmn.Contains(t.Name+t.Symbol, token) {
						tn := t.Address.String()
						if t.Unique {
							tn = t.Symbol
						}

						options = append(options, ui.ACOption{
							Name:   fmt.Sprintf("%-6s %s", t.Symbol, t.GetPrintName()),
							Result: command + " " + subcommand + " " + strconv.Itoa(b.ChainId) + " " + tn + " "})
					}
				}
			}
			return "token", &options, token
		}
	case 4:
		if subcommand == "set_feeder" {
			b := w.GetBlockchainByName(bchain)

			if b != nil && p[4] == "" {
				t := w.GetToken(b.ChainId, token)
				if t != nil {
					for _, f := range cmn.KNOWN_FEEDERS {
						if cmn.Contains(f, p[4]) {
							options = append(options, ui.ACOption{
								Name:   f,
								Result: command + " set_feeder " + strconv.Itoa(b.ChainId) + " '" + token + "' '" + f + "' "})
						}
					}
					return "feeder", &options, p[4]

				}
			}
		}
	}

	return "", &options, ""
}

func Price_Process(c *Command, input string) {
	w := cmn.CurrentWallet
	if w == nil {
		ui.PrintErrorf("No wallet open")
		return
	}

	p := cmn.SplitN(input, 6)
	subcommand, bchain, token := p[1], p[2], p[3]

	switch subcommand {
	case "list", "":
		list_price_feeders(w)
	case "discover":
		if token == "" {
			ui.Printf("USAGE: discover [CHAIN] [TOKEN_ADDR]\n")
			return
		}

		b := w.GetBlockchainByName(bchain)
		if b == nil {
			ui.PrintErrorf("Invalid blockchain")
			return
		}

		t := w.GetToken(b.ChainId, token)
		if t == nil {
			ui.PrintErrorf("Invalid token address")
			return
		}

		ui.Printf("\nDiscovering trading pairs for token\n")
		ui.Printf("Token name: %s\n", t.GetPrintName())
		if !t.Native {
			ui.Printf("Token Address: %s\n", t.Address.Hex())
		}
		ui.Printf("Feeder type: %s\n", t.PriceFeeder)
		ui.Printf("Feeder Param: %s\n", t.PriceFeedParam)

		pi_list, err := price.GetPriceInfoList(b.ChainId, t.Address.Hex())
		if err != nil {
			ui.PrintErrorf("Error discovering trading pairs: %v", err)
			return
		}

		if len(pi_list) == 0 {
			ui.Printf("No trading pairs found\n")
			return
		}

		ui.Printf("\n   Feeder          Pair    Liquidity    Price       ID\n")

		for _, p := range pi_list {
			ui.Printf("%-14s ", p.PriceFeeder)
			ui.Printf(" %4s/%-5s ", p.BaseToken, p.QuoteToken)
			cmn.AddDollarLink(ui.Terminal.Screen, p.Liquidity)
			cmn.AddDollarLink(ui.Terminal.Screen, p.PriceUsd)

			tn := t.Address.String()
			if t.Unique {
				tn = t.Symbol
			}

			ui.Terminal.Screen.AddLink(cmn.ICON_FEED, "command price set_feeder '"+bchain+"' "+
				tn+" "+p.PriceFeeder+" '"+p.PairID+"'",
				"Connect price feeder to trading pair", "")
			ui.Terminal.Screen.AddLink(cmn.ICON_LINK, "open "+p.URL, p.URL, "")

			ui.Printf(" %s ", p.PairID)

			ui.Printf("\n")

		}
	case "set_feeder":
		bchain, token, feeder, param := p[2], p[3], p[4], p[5]

		b := w.GetBlockchainByName(bchain)
		if b == nil {
			ui.PrintErrorf("Invalid blockchain")
			return
		}

		t := w.GetToken(b.ChainId, token)
		if t == nil {
			ui.PrintErrorf("Invalid token address")
			return
		}

		if !cmn.IsInArray(cmn.KNOWN_FEEDERS, feeder) {
			ui.PrintErrorf("Invalid feeder")
			return
		}

		if param == "" {
			ui.PrintErrorf("Invalid feeder param")
			return
		}

		t.PriceFeeder = feeder
		t.PriceFeedParam = param

		if t.Native {
			wt := w.GetToken(b.ChainId, b.WTokenAddress.Hex())
			if wt != nil {
				wt.PriceFeeder = feeder
				wt.PriceFeedParam = param
			}
		} else {
			if t.Address.Cmp(b.WTokenAddress) != 0 {
				nt, _ := w.GetNativeToken(b)
				if nt != nil {
					nt.PriceFeeder = feeder
					nt.PriceFeedParam = param
				}
			}
		}

		err := w.Save()
		if err != nil {
			ui.PrintErrorf("Error saving wallet: %v", err)
			return
		}

		ui.Printf("Price feeder set for trading pair\n")

	case "update":
		err := price.Update(w)
		if err != nil {
			ui.PrintErrorf("Error updating price feeders: %v", err)
			return
		}

		bus.Send("ui", "notify", "Price feeders updated")

	default:
		ui.PrintErrorf("Invalid subcommand: %s", subcommand)

	}
}

func list_price_feeders(w *cmn.Wallet) {

	sort.Slice(w.Tokens, func(i, j int) bool {
		if w.Tokens[i].PriceFeeder == w.Tokens[j].PriceFeeder {
			return w.Tokens[i].PriceFeedParam < w.Tokens[j].PriceFeedParam
		} else {
			return w.Tokens[i].PriceFeeder < w.Tokens[j].PriceFeeder
		}
	})

	for _, t := range w.Tokens {
		if t.PriceFeeder == "" {
			continue
		}

		b := w.GetBlockchain(t.ChainId)
		if b == nil {
			log.Error().Msgf("Blockchain not found: %d", t.ChainId)
			continue
		}

		sp := t.PriceFeedParam
		if len(sp) > 12 {
			sp = sp[:9] + "..."
		}

		url := price.GetUrl(t.PriceFeeder, t.ChainId, t.PriceFeedParam)

		ui.Terminal.Screen.AddLink(
			fmt.Sprintf("%-14s %-12s", t.PriceFeeder, sp),
			"open "+url,
			url, "")

		ui.Printf(" %-8s ", t.Symbol)

		if t.Price != 0. {
			cmn.AddDollarLink(ui.Terminal.Screen, t.Price)
		} else {
			ui.Printf("          ")
		}

		ui.Terminal.Screen.AddLink(cmn.ICON_EDIT, "command token edit "+strconv.Itoa(t.ChainId)+" "+t.Address.String()+" ", "Edit token", "")

		ui.Terminal.Screen.AddLink(cmn.ICON_FEED, "command p discover '"+b.Name+"' '"+t.Address.String()+"'", "Discover price", "")

		if t.Native {
			ui.Printf("Native     ")
		} else {
			cmn.AddAddressShortLink(ui.Terminal.Screen, t.Address)
		}

		ui.Printf(" %-12s %-s\n", b.Name, t.Name)
	}

	ui.Printf("\n")

}
