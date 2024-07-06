package command

import (
	"math/big"
	"sort"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/eth"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
	"github.com/ethereum/go-ethereum/common"
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

	if wallet.CurrentWallet == nil {
		return "", nil, ""
	}

	w := wallet.CurrentWallet

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
					Name:   t.Symbol,
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
		for _, chain := range w.Blockchains {
			if cmn.Contains(chain.Name, param) {
				options = append(options, ui.ACOption{
					Name: chain.Name, Result: command + " " + subcommand + " '" + chain.Name + "' "})
			}
		}
		return "blockchain", &options, param
	}

	return "", &options, ""
}

func Token_Process(c *Command, input string) {
	if wallet.CurrentWallet == nil {
		ui.PrintErrorf("\nNo wallet open\n")
		return
	}

	w := wallet.CurrentWallet

	//parse command subcommand parameters
	p := cmn.SplitN(input, 5)
	//execute command
	subcommand := p[1]

	switch subcommand {
	case "add":

		chain := p[2]
		address := p[3]

		if chain == "" {
			ui.PrintErrorf("\nUsage: token add [BLOCKCHAIN] [ADDRESS]\n")
			return
		}

		if address == "" {
			ui.PrintErrorf("\nUsage: token add %s [ADDRESS]\n", chain)
			return
		}

		if !common.IsHexAddress(address) {
			ui.PrintErrorf("\nInvalid address: %s\n", address)
			return
		}

		addr := common.HexToAddress(address)
		bchain := w.GetBlockchain(chain)
		if bchain == nil {
			ui.PrintErrorf("\nBlockchain not found: %s\n", chain)
			return
		}

		if w.GetTokenByAddress(bchain.Name, addr) != nil {
			ui.PrintErrorf("\nToken already exists: %s\n", address)
			return
		}

		symbol, name, decimals, err := eth.GetERC20TokenInfo(bchain, &addr)
		if err != nil {
			ui.PrintErrorf("\nError getting token info: %v\n", err)
			return
		}

		t := &cmn.Token{
			Blockchain: bchain.Name,
			Name:       name,
			Symbol:     symbol,
			Address:    addr,
			Decimals:   decimals,
		}

		w.Tokens = append(w.Tokens, t)
		w.Save()

		ui.Printf("\nToken added: %s %s\n", symbol, addr.String())
	case "remove":
		chain := p[2]
		address := p[3]

		if chain == "" {
			ui.PrintErrorf("\nUsage: token remove [BLOCKCHAIN] [ADDRESS]\n")
			return
		}

		if address == "" {
			ui.PrintErrorf("\nUsage: token remove %s [ADDRESS]\n", chain)
			return
		}

		if !common.IsHexAddress(address) {
			ui.PrintErrorf("\nInvalid address: %s\n", address)
			return
		}

		addr := common.HexToAddress(address)

		t := w.GetTokenByAddress(chain, addr)
		if t == nil {
			ui.PrintErrorf("\nToken not found: %s\n", address)
			return
		}

		ui.Gui.ShowPopup(ui.DlgConfirm(
			"Remove address",
			`
<c>Are you sure you want to remove token:
<c> `+t.Name+`
<c> `+t.Symbol+"? \n",
			func() {
				w.DeleteToken(chain, addr)
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
			ui.Printf("  %-8s %-10s ", t.Symbol, t.Blockchain)

			if t.Native {
				ui.Printf("Native")
			} else {
				ui.AddAddressShortLink(ui.Terminal.Screen, t.Address)
				ui.Printf(" ")
				ui.Terminal.Screen.AddLink(gocui.ICON_DELETE, "command token remove '"+t.Blockchain+"' '"+t.Address.String()+"'", "Remove token", "")
				ui.Printf(" %s", t.Name)
			}
			ui.Printf("\n")
		}

		ui.Printf("\n")
	case "balance":
		chain := p[2]
		token := p[3]
		address := p[4]

		if chain == "" {
			ui.PrintErrorf("\nUsage: token balance [BLOCKCHAIN] [TOKEN/ADDRESS] [ADDRESS]\n")
			return
		}

		bchain := w.GetBlockchain(chain)
		if bchain == nil {
			ui.PrintErrorf("\nBlockchain not found: %s\n", chain)
			return
		}

		if token == "" {
			ui.PrintErrorf("\nUsage: token balance %s [TOKEN/ADDRESS] [ADDRESS]\n", chain)
			return
		}

		t := w.GetTokenBySymbol(chain, token)
		if t == nil {
			t = w.GetTokenByAddress(chain, common.HexToAddress(token))
		}

		if t == nil {
			ui.PrintErrorf("\nToken not found (or ambiguous): %s\n", token)
			return
		}

		for _, a := range w.Addresses {
			if address != "" && a.Address.String() != address {
				continue
			}

			balance, err := eth.BalanceOf(bchain, t, a.Address)
			if err != nil {
				ui.PrintErrorf("\nError getting balance: %v\n", err)
				return
			}

			if balance.Cmp(big.NewInt(0)) != 0 {

				tid := t.Symbol
				if !t.Unique {
					tid = t.Address.String()
				}

				ui.AddAddressShortLink(ui.Terminal.Screen, a.Address)
				ui.Printf(" ")
				ui.AddValueSymbolLink(ui.Terminal.Screen, balance, t)
				ui.Printf(" %s ", a.Name)
				ui.Terminal.Screen.AddLink(gocui.ICON_SEND, "command send '"+chain+"' '"+tid+"' '"+a.Address.String()+"'", "Send tokens", "")
				ui.Printf("\n")
			}
		}

	default:
		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
	}

}
