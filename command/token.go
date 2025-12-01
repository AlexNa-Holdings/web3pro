package command

import (
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/eth"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

var token_subcommands = []string{"on", "off", "remove", "edit", "add", "balance", "list"}

func NewTokenCommand() *Command {
	return &Command{
		Command:      "token",
		ShortCommand: "t",
		Usage: `
Usage: token [COMMAND]

Manage tokens

Commands:
  on                            - Show tokens panel
  off                           - Hide tokens panel
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

	last_param := len(p) - 1
	for last_param > 0 && p[last_param] == "" {
		last_param--
	}

	if strings.HasSuffix(input, " ") {
		last_param++
	}

	switch last_param {
	case 0, 1:

		if !cmn.IsInArray(token_subcommands, subcommand) {
			for _, sc := range token_subcommands {
				if input == "" || strings.Contains(sc, subcommand) {
					options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
				}
			}
			return "action", &options, subcommand
		}

	case 2:
		if subcommand == "list" || subcommand == "add" || subcommand == "remove" || subcommand == "balance" || subcommand == "edit" {
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

	case 3:
		b := w.GetBlockchainByName(param)
		token := p[3]

		if subcommand == "balance" && b != nil &&
			(token == "" || (w.GetTokenByAddress(b.ChainId, common.HexToAddress(token)) == nil && w.GetTokenBySymbol(b.ChainId, token) == nil)) {
			for _, t := range w.Tokens {
				if t.ChainId != b.ChainId {
					continue
				}
				if cmn.Contains(t.Symbol, token) || cmn.Contains(t.Address.String(), token) || cmn.Contains(t.Name, token) {

					id := t.Symbol
					if !t.Unique {
						id = t.Address.String()
					}

					options = append(options, ui.ACOption{
						Name:   fmt.Sprintf("%-6s %s", t.Symbol, t.GetPrintName()),
						Result: fmt.Sprintf("%s balance %d %s", command, b.ChainId, id)})
				}
			}
			return "token", &options, token
		}
	case 4:
		b := w.GetBlockchainByName(param)
		t := w.GetToken(b.ChainId, p[3])

		if subcommand == "balance" && b != nil && t != nil {
			for _, a := range w.Addresses {
				if cmn.Contains(a.Address.String(), p[4]) || cmn.Contains(a.Name, param) {
					options = append(options, ui.ACOption{
						Name:   cmn.ShortAddress(a.Address) + " " + a.Name,
						Result: fmt.Sprintf("%s balance %s %s %s", command, p[2], p[3], a.Address.String())})
				}
			}
			return "address", &options, param
		}
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
	case "on":
		w.TokenPaneOn = true
		w.Save()
	case "off":
		w.TokenPaneOn = false
		w.Save()
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
		bchain := w.GetBlockchainByName(chain)
		if bchain == nil {
			ui.PrintErrorf("Blockchain not found: %s", chain)
			return
		}

		if w.GetTokenByAddress(bchain.ChainId, addr) != nil {
			ui.PrintErrorf("Token already exists: %s", address)
			return
		}

		symbol, name, decimals, err := eth.GetERC20TokenInfo(bchain, addr)
		if err != nil {
			ui.PrintErrorf("Error getting token info: %v", err)
			return
		}

		err = w.AddToken(bchain.ChainId, addr, name, symbol, decimals)
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

		b := w.GetBlockchainByName(chain)
		if b == nil {
			chain_id, err := strconv.Atoi(chain)
			if err != nil {
				ui.PrintErrorf("Invalid blockchain: %s", chain)
				return
			}
			b := w.GetBlockchain(chain_id)
			if b == nil {
				ui.PrintErrorf("token_process: Blockchain not found: %s", chain)
				return
			}
		}

		t := w.GetTokenByAddress(b.ChainId, addr)
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
			func() bool {
				w.DeleteToken(b.ChainId, addr)
				ui.Notification.Show("Token removed")
				return true
			}))
	case "list", "":

		chain := p[2]

		lb := w.GetBlockchainByName(chain)

		ui.Printf("\nTokens:\n")

		// Sort the tokens by Blockchain and Symbol
		sort.Slice(w.Tokens, func(i, j int) bool {
			if w.Tokens[i].ChainId == w.Tokens[j].ChainId {
				return w.Tokens[i].Symbol < w.Tokens[j].Symbol
			}
			return w.Tokens[i].ChainId < w.Tokens[j].ChainId
		})

		for _, t := range w.Tokens {
			if chain != "" && lb != nil && t.ChainId != lb.ChainId {
				continue
			}

			b := w.GetBlockchain(t.ChainId)
			if b == nil {
				log.Error().Msgf("Blockchain not found: %d", t.ChainId)
				continue
			}

			ui.Printf("%-8s ", t.Symbol)

			if t.Price != 0. {
				cmn.AddFixedDollarLink(ui.Terminal.Screen, t.Price, 10)
			} else {
				ui.Printf("%10s", "")
			}

			if t.PriceChange24 > 0 {
				ui.Printf(ui.F(gocui.ColorGreen)+"\uf0d8%6.2f%% "+ui.F(ui.Terminal.Screen.FgColor), t.PriceChange24)
			} else if t.PriceChange24 < 0 {
				ui.Printf(ui.F(gocui.ColorRed)+"\uf0d7%6.2f%% "+ui.F(ui.Terminal.Screen.FgColor), -t.PriceChange24)
			} else {
				ui.Printf("%9s", "")
			}

			ui.Terminal.Screen.AddLink(cmn.ICON_EDIT, "command token edit "+strconv.Itoa(t.ChainId)+" "+t.Address.String()+" ", "Edit token", "")

			if !t.Native {
				ui.Terminal.Screen.AddLink(cmn.ICON_LINK, "open "+b.ExplorerLink(t.Address), b.ExplorerLink(t.Address), "")
				ui.Terminal.Screen.AddLink(cmn.ICON_DELETE, "command token remove "+strconv.Itoa(t.ChainId)+" '"+t.Address.String()+"'", "Remove token", "")
			} else {
				ui.Printf("    ")
			}

			ui.Terminal.Screen.AddLink(cmn.ICON_FEED, "command p discover '"+b.Name+"' '"+t.Address.String()+"'", "Discover price", "")

			if t.Native {
				ui.Printf("%-15s", "Native")
			} else {
				cmn.AddFixedAddressShortLink(ui.Terminal.Screen, t.Address, 15)
			}

			ui.Printf(" %-25s %s\n", b.Name, t.Name)
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

		b := w.GetBlockchainByName(chain)
		if b == nil {
			chain_id, err := strconv.Atoi(chain)
			if err != nil {
				ui.PrintErrorf("Invalid blockchain: %s", chain)
				return
			}
			b := w.GetBlockchain(chain_id)
			if b == nil {
				ui.PrintErrorf("Blockchain not found: %s", chain)
				return
			}
		}

		if token == "" {
			ui.PrintErrorf("Usage: token balance %s [TOKEN/ADDRESS] [ADDRESS]", chain)
			return
		}

		t := w.GetTokenBySymbol(b.ChainId, token)
		if t == nil {
			t = w.GetTokenByAddress(b.ChainId, common.HexToAddress(token))
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

			balance, err := eth.BalanceOf(b, t, a.Address)
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
				cmn.ICON_SEND,
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
	case "edit":
		chain := p[2]
		token := p[3]

		b := w.GetBlockchainByName(chain)
		if b == nil {
			chain_id, err := strconv.Atoi(chain)
			if err != nil {
				ui.PrintErrorf("Invalid blockchain: %s", chain)
				return
			}
			b := w.GetBlockchain(chain_id)
			if b == nil {
				ui.PrintErrorf("Blockchain not found: %s", chain)
				return
			}
		}

		if token == "" {
			ui.PrintErrorf("Usage: token balance %s [TOKEN/ADDRESS] [ADDRESS]", chain)
			return
		}

		t := w.GetTokenBySymbol(b.ChainId, token)
		if t == nil {
			t = w.GetTokenByAddress(b.ChainId, common.HexToAddress(token))
		}

		if t == nil {
			ui.PrintErrorf("Token not found (or ambiguous): %s", token)
			return
		}

		bus.Send("ui", "popup", ui.DlgTokenEdit(t))

	default:
		ui.PrintErrorf("Invalid subcommand: %s", subcommand)
	}

}
