package command

import (
	"fmt"
	"sort"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/ethereum/go-ethereum/common"
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
				for _, lp := range cmn.PrefedinedLP_V3[b.ChainID] {
					options = append(options, ui.ACOption{
						Name:   lp.Name,
						Result: command + " " + subcommand + " '" + b.Name + "' '" + lp.Address.Hex() + "' '" + lp.Name + "'"})
				}

				return "address", &options, addr
			}
		}

		if subcommand == "discover" {
			for _, lp := range w.LP_V3_Providers {
				if cmn.Contains(lp.Name, addr) {
					options = append(options, ui.ACOption{
						Name:   lp.Name,
						Result: command + " " + subcommand + " '" + lp.Blockchain + "' '" + lp.Name + "'"})
				}
			}
			return "name", &options, addr
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

	p := cmn.SplitN(input, 5)
	_, subcommand, chain, addr, name := p[0], p[1], p[2], p[3], p[4]

	switch subcommand {
	case "list", "":

	case "providers":
		ui.Printf("\nV3 Providers\n\n")

		if len(w.LP_V3_Providers) == 0 {
			ui.Printf("(no providers)\n")
		}

		sort.Slice(w.LP_V3_Providers, func(i, j int) bool {
			if w.LP_V3_Providers[i].Blockchain == w.LP_V3_Providers[j].Blockchain {
				return w.LP_V3_Providers[i].Name < w.LP_V3_Providers[j].Name
			}
			return w.LP_V3_Providers[i].Blockchain < w.LP_V3_Providers[j].Blockchain
		})

		for i, lp := range w.LP_V3_Providers {
			ui.Printf("%d %12s %s ", i+1, lp.Blockchain, lp.Name)
			ui.Terminal.Screen.AddLink(gocui.ICON_EDIT, "command lp_v3 edit '"+lp.Blockchain+"' '"+lp.Address.Hex()+"' '"+lp.Name+"'", "Edit provider", "")
			ui.Terminal.Screen.AddLink(gocui.ICON_DELETE, "command lp_v3 remove '"+lp.Blockchain+"' '"+lp.Address.Hex()+"'", "Remove provider", "")
		}

		ui.Printf("\n")

	case "add":
		b := w.GetBlockchain(chain)
		if b == nil {
			err = fmt.Errorf("blockchain not found: %s", chain)
			break
		}

		bus.Send("ui", "popup", ui.DlgLP_V3_add(b, addr, name))
	case "edit":
		b := w.GetBlockchain(chain)
		if b == nil {
			err = fmt.Errorf("blockchain not found: %s", chain)
			break
		}

		bus.Send("ui", "popup", ui.DlgLP_V3_edit(b, addr, name))
	case "remove":
		b := w.GetBlockchain(chain)
		if b == nil {
			err = fmt.Errorf("blockchain not found: %s", chain)
			break
		}

		a := common.HexToAddress(addr)
		if a == (common.Address{}) {
			err = fmt.Errorf("invalid address: %s", addr)
			break
		}

		lp := w.GetLP_V3(b.ChainID, a)
		if lp == nil {
			err = fmt.Errorf("provider not found: %s", addr)
			break
		}

		bus.Send("ui", "popup", ui.DlgConfirm(
			"Remove provider",
			`	
<c>Are you sure you want to remove provider?</c>

       Name:`+lp.Name+`
 Blockchain:`+lp.Blockchain+`
    Address:`+lp.Address.String()+`
`,
			func() {
				err := w.RemoveLP_V3(b.ChainID, a)
				if err != nil {
					ui.PrintErrorf("Error removing provider: %v", err)
					return
				}
				ui.Notification.Show("Provider removed")
			}))

	case "discover":
		resp := bus.Fetch("lp_v3", "discover", bus.B_LP_V3_Discover{
			Chain: chain,
			Name:  name,
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
