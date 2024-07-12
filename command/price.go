package command

import (
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/price"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/ethereum/go-ethereum/common"
)

var price_subcommands = []string{"set_feeder", "discover", "update"}

func NewPriceCommand() *Command {
	return &Command{
		Command:      "price",
		ShortCommand: "p",
		Usage: `
Usage: 

  discover [BLOCKCHAIN] [TOKEN_ADDR]                   - Discover trading pairs for token
  set_feeder [BLOCKCHAIN] [PAIR_ADDR] [FEEDER] [PARAM} - Set price feeder for trading pair

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
	p := cmn.SplitN(input, 5)
	command, subcommand, bchain, token := p[0], p[1], p[2], p[3]

	if !cmn.IsInArray(price_subcommands, subcommand) {
		for _, sc := range price_subcommands {
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	}

	if subcommand == "discover" {

		b := w.GetBlockchain(bchain)

		if token == "" && b == nil {
			for _, b := range w.Blockchains {
				if cmn.Contains(b.Name, bchain) {
					options = append(options, ui.ACOption{Name: b.Name, Result: command + " " + subcommand + " '" + b.Name + "'"})
				}
			}
			return "blockchain", &options, bchain
		}

		if b != nil && (token == "" || !strings.HasSuffix(input, " ")) {
			for _, t := range w.Tokens {
				if t.Blockchain == bchain && cmn.Contains(t.Name+t.Symbol, token) {
					tn := t.Address.String()
					if t.Unique {
						tn = t.Symbol
					}

					options = append(options, ui.ACOption{
						Name:   fmt.Sprintf("%-6s %s", t.Symbol, t.GetPrintName()),
						Result: command + " discover '" + bchain + "' " + tn + " "})
				}
			}
		}
		return "token", &options, token
	}

	return "", &options, ""
}

func Price_Process(c *Command, input string) {
	if cmn.CurrentWallet == nil {
		return
	}

	w := cmn.CurrentWallet

	p := cmn.SplitN(input, 6)
	subcommand, bchain, token := p[1], p[2], p[3]

	switch subcommand {
	case "discover", "":
		if token == "" {
			ui.Printf("USAGE: discover [TOKEN_ADDR]\n")
			return
		}

		b := w.GetBlockchain(bchain)
		if b == nil {
			ui.PrintErrorf("Invalid blockchain\n")
			return
		}

		t := w.GetToken(bchain, token)
		if t == nil {
			ui.PrintErrorf("Invalid token address\n")
			return
		}

		ui.Printf("\nDiscovering trading pairs for token\n")
		ui.Printf("Token name: %s\n", t.GetPrintName())
		a := t.Address
		if t.Native {
			a = common.HexToAddress(b.WTokenAddress.Hex())
			ui.Printf("Wrapped Token Address: %s\n", a.Hex())
		} else {
			ui.Printf("Token Address: %s\n", a.Hex())
		}
		ui.Printf("Feeder type: %s\n", t.PriceFeeder)
		ui.Printf("Feeder Param: %s\n", t.PriceFeedParam)

		pairs, err := price.GetPairs(b.ChainId, a.Hex())
		if err != nil {
			ui.PrintErrorf("Error discovering trading pairs: %v\n", err)
			return
		}

		ui.Printf("\n   Feeder      Pair Addr    Liquidity  Price\n")

		for i, p := range pairs {
			ui.Printf("%2d %10s ", i+1, p.PriceFeeder)
			ui.AddAddressShortLink(ui.Terminal.Screen, common.Address(common.FromHex(p.PairAddress)))
			ui.Printf(" %s", cmn.FmtFloat64D(p.Liquidity, true))
			ui.Printf(" %s", cmn.FmtFloat64D(p.PriceUsd, true))
			ui.Printf(" %s/%s ", p.BaseToken, p.QuoteToken)

			tn := t.Address.String()
			if t.Unique {
				tn = t.Symbol
			}

			ui.Terminal.Screen.AddLink(gocui.ICON_FEED, "command price set_feeder '"+bchain+"' "+
				tn+" "+p.PriceFeeder+" '"+p.PairAddress+"'",
				"Connect price feeder to trading pair", "")

			ui.Printf("\n")

		}
	case "set_feeder":
		bchain, token, feeder, param := p[2], p[3], p[4], p[5]

		b := w.GetBlockchain(bchain)
		if b == nil {
			ui.PrintErrorf("Invalid blockchain\n")
			return
		}

		t := w.GetToken(bchain, token)
		if t == nil {
			ui.PrintErrorf("Invalid token address\n")
			return
		}

		if !cmn.IsInArray(price.KNOWN_FEEDERS, feeder) {
			ui.PrintErrorf("Invalid feeder\n")
			return
		}

		if param == "" {
			ui.PrintErrorf("Invalid feeder param\n")
			return
		}

		t.PriceFeeder = feeder
		t.PriceFeedParam = param
		err := w.Save()
		if err != nil {
			ui.PrintErrorf("Error saving wallet: %v\n", err)
			return
		}

		ui.Printf("Price feeder set for trading pair\n")

	case "update":
		err := price.Update(w)
		if err != nil {
			ui.PrintErrorf("Error updating price feeders: %v\n", err)
			return
		}

		cmn.Notify("Price feeders updated")

	default:
		ui.PrintErrorf("Invalid subcommand: %s\n", subcommand)

	}
}
