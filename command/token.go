package command

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/eth"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

var token_subcommands = []string{"remove", "add", "balance", "list"}

func NewTokenCommand() *Command {
	return &Command{
		Command:      "token",
		ShortCommand: "t",
		Usage: `
Usage: token [COMMAND]

Manage tokens

Commands:
  add [BLOCKCHAIN] [ADDRESS]    - Add new token
  list [BLOCKCHAIN]             - List tokens
  remove [BLOCKCHAIN] [ADDRESS] - Remove token  
  balance [BLOCKCHAIN] [TOKEN/ADDRESS] [ADDRESS] - Get token balance
		`,
		Help:             `Manage tokens`,
		Process:          Token_Process,
		AutoCompleteFunc: Token_AutoComplete,
	}
}

func Token_AutoComplete(input string) (string, *[]ui.ACOption, string) {

	if cmn.CurrentWallet == nil {
		return "", nil, ""
	}

	w := cmn.CurrentWallet

	options := []ui.ACOption{}
	p := cmn.SplitN(input, 5)
	command, subcommand, param := p[0], p[1], p[2]

	if !cmn.IsInArray(token_subcommands, subcommand) {
		for _, sc := range token_subcommands {
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	}

	b := w.GetBlockchain(param)
	token := p[3]

	if subcommand == "balance" && b != nil &&
		(token == "" || (w.GetTokenByAddress(b.Name, common.HexToAddress(token)) == nil && w.GetTokenBySymbol(b.Name, token) == nil)) {
		for _, t := range w.Tokens {
			if t.Blockchain != b.Name {
				continue
			}
			if cmn.Contains(t.Symbol, token) || cmn.Contains(t.Address.String(), token) || cmn.Contains(t.Name, token) {

				id := t.Symbol
				if !t.Unique {
					id = t.Address.String()
				}

				options = append(options, ui.ACOption{
					Name:   fmt.Sprintf("%-6s %s", t.Symbol, t.GetPrintName()),
					Result: command + " balance '" + b.Name + "' " + id + " "})
			}
		}
		return "token", &options, token
	}

	adr := p[4]

	if subcommand == "balance" && b != nil &&
		(w.GetTokenByAddress(b.Name, common.HexToAddress(token)) != nil || w.GetTokenBySymbol(b.Name, token) != nil) {
		for _, a := range w.Addresses {
			if cmn.Contains(a.Name+a.Address.String(), adr) {
				options = append(options, ui.ACOption{
					Name:   cmn.ShortAddress(a.Address) + " " + a.Name,
					Result: command + " balance '" + b.Name + "' " + token + " " + a.Address.String()})
			}
		}
		return "address", &options, adr
	}

	if subcommand == "list" || subcommand == "add" || subcommand == "remove" || subcommand == "balance" {
		if param == "" || !strings.HasSuffix(input, " ") {
			for _, chain := range w.Blockchains {
				if cmn.Contains(chain.Name, param) {
					options = append(options, ui.ACOption{
						Name: chain.Name, Result: command + " " + subcommand + " '" + chain.Name + "' "})
				}
			}
		}
		return "blockchain", &options, param
	}

	return "", &options, ""
}

func Token_Process(c *Command, input string) {
	if cmn.CurrentWallet == nil {
		ui.PrintErrorf("No wallet open")
		return
	}

	w := cmn.CurrentWallet

	//parse command subcommand parameters
	p := cmn.SplitN(input, 5)
	//execute command
	subcommand := p[1]

	switch subcommand {
	case "add":

		chain := p[2]
		address := p[3]

		if chain == "" {
			ui.PrintErrorf("Usage: token add [BLOCKCHAIN] [ADDRESS]")
			return
		}

		if address == "" {
			ui.PrintErrorf("Usage: token add %s [ADDRESS]", chain)
			return
		}

		if !common.IsHexAddress(address) {
			ui.PrintErrorf("Invalid address: %s", address)
			return
		}

		addr := common.HexToAddress(address)
		bchain := w.GetBlockchain(chain)
		if bchain == nil {
			ui.PrintErrorf("Blockchain not found: %s", chain)
			return
		}

		if w.GetTokenByAddress(bchain.Name, addr) != nil {
			ui.PrintErrorf("Token already exists: %s", address)
			return
		}

		symbol, name, decimals, err := eth.GetERC20TokenInfo(bchain, addr)
		if err != nil {
			ui.PrintErrorf("Error getting token info: %v", err)
			return
		}

		err = w.AddToken(bchain.Name, addr, name, symbol, decimals)
		if err != nil {
			ui.PrintErrorf("Error adding token: %v", err)
			return
		}

		ui.Printf("\nToken added: %s %s\n", symbol, addr.String())
	case "remove":
		chain := p[2]
		address := p[3]

		if chain == "" {
			ui.PrintErrorf("Usage: token remove [BLOCKCHAIN] [ADDRESS]")
			return
		}

		if address == "" {
			ui.PrintErrorf("Usage: token remove %s [ADDRESS]", chain)
			return
		}

		if !common.IsHexAddress(address) {
			ui.PrintErrorf("Invalid address: %s", address)
			return
		}

		addr := common.HexToAddress(address)

		t := w.GetTokenByAddress(chain, addr)
		if t == nil {
			ui.PrintErrorf("Token not found: %s", address)
			return
		}

		bus.Send("ui", "popup", ui.DlgConfirm(
			"Remove address",
			`
<c>Are you sure you want to remove token:
<c> `+t.Name+`
<c> `+t.Symbol+"? \n",
			func() {
				w.DeleteToken(chain, addr)
				w.MarkUniqueTokens()
				w.Save()
				ui.Notification.Show("Token removed")
			}))
	case "list", "":

		chain := p[2]

		ui.Printf("\nTokens:\n")

		// Sort the tokens by Blockchain and Symbol
		sort.Slice(w.Tokens, func(i, j int) bool {
			if w.Tokens[i].Blockchain == w.Tokens[j].Blockchain {
				return w.Tokens[i].Symbol < w.Tokens[j].Symbol
			}
			return w.Tokens[i].Blockchain < w.Tokens[j].Blockchain
		})

		for _, t := range w.Tokens {
			if chain != "" && t.Blockchain != chain {
				continue
			}

			price := "          "
			if t.Price != 0. {
				price = cmn.FmtFloat64D(t.Price, true)
			}

			b := w.GetBlockchain(t.Blockchain)
			if b == nil {
				log.Error().Msgf("Blockchain not found: %s", t.Blockchain)
				continue
			}

			ui.Printf("%-8s %s ", t.Symbol, price)

			if t.Native {
				ui.Printf("Native          ")
			} else {
				cmn.AddAddressShortLink(ui.Terminal.Screen, t.Address)
				ui.Printf(" ")
				ui.Terminal.Screen.AddLink(gocui.ICON_LINK, "open "+b.ExplorerLink(t.Address), b.ExplorerLink(t.Address), "")
				ui.Terminal.Screen.AddLink(gocui.ICON_DELETE, "command token remove '"+t.Blockchain+"' '"+t.Address.String()+"'", "Remove token", "")
			}
			ui.Printf("%s | %s", t.Blockchain, t.Name)
			ui.Printf("\n")
		}

		ui.Printf("\n")
	case "balance":
		chain := p[2]
		token := p[3]
		address := p[4]

		if chain == "" {
			ui.PrintErrorf("Usage: token balance [BLOCKCHAIN] [TOKEN/ADDRESS] [ADDRESS]")
			return
		}

		bchain := w.GetBlockchain(chain)
		if bchain == nil {
			ui.PrintErrorf("Blockchain not found: %s", chain)
			return
		}

		if token == "" {
			ui.PrintErrorf("Usage: token balance %s [TOKEN/ADDRESS] [ADDRESS]", chain)
			return
		}

		t := w.GetTokenBySymbol(chain, token)
		if t == nil {
			t = w.GetTokenByAddress(chain, common.HexToAddress(token))
		}

		if t == nil {
			ui.PrintErrorf("Token not found (or ambiguous): %s", token)
			return
		}

		tid := t.Symbol
		if !t.Unique {
			tid = t.Address.String()
		}

		blist := make([]struct {
			Address *cmn.Address
			Balance *big.Int
		}, 0)

		ui.Printf("\nToken: %s %s\n", t.Symbol, t.Name)

		if t.Price > 0. {
			ui.Printf("Price: %s\n", cmn.FmtFloat64D(t.Price, true))
		}

		for _, a := range w.Addresses {
			if address != "" && a.Address.String() != address {
				continue
			}

			balance, err := eth.BalanceOf(bchain, t, a.Address)
			if err != nil {
				ui.PrintErrorf("Error getting balance: %v", err)
				return
			}

			if balance.Cmp(big.NewInt(0)) != 0 {

				blist = append(blist, struct {
					Address *cmn.Address
					Balance *big.Int
				}{Address: a, Balance: balance})
			}
		}

		//sort by balance
		sort.Slice(blist, func(i, j int) bool {
			return blist[i].Balance.Cmp(blist[j].Balance) > 0
		})

		total_balance := big.NewInt(0)
		total_dollars := float64(0)
		for _, b := range blist {
			cmn.AddAddressShortLink(ui.Terminal.Screen, b.Address.Address)
			ui.Printf(" ")
			cmn.AddValueSymbolLink(ui.Terminal.Screen, b.Balance, t)
			ui.Printf(" ")
			if t.Price > 0. {
				cmn.AddDollarValueLink(ui.Terminal.Screen, b.Balance, t)
			}
			ui.Printf(" %s ", b.Address.Name)
			ui.Terminal.Screen.AddLink(
				gocui.ICON_SEND,
				"command send '"+chain+"' '"+tid+"' "+string(b.Address.Address.String()),
				"Send tokens",
				"")

			ui.Printf("\n")
			total_balance.Add(total_balance, b.Balance)
			total_dollars += t.Price * t.Float64(b.Balance)
		}

		ui.Printf("\n  Total: ")
		cmn.AddValueSymbolLink(ui.Terminal.Screen, total_balance, t)
		ui.Printf("\n")
		if t.Price > 0. {
			ui.Printf("Total $: ")
			cmn.AddDollarValueLink(ui.Terminal.Screen, total_balance, t)
			ui.Printf("\n")
		}

	default:
		ui.PrintErrorf("Invalid subcommand: %s", subcommand)
	}

}
