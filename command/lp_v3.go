package command

import (
	"fmt"
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
	"on", "off", "add", "edit", "remove", "discover", "providers",
	"list",
}

var Q128, _ = new(big.Int).SetString("100000000000000000000000000000000", 16)
var TWO96 = new(big.Int).Exp(big.NewInt(2), big.NewInt(96), nil)

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
  edit [CHAIN] [ADDR]       - Edit v3 provider
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
		if subcommand == "add" || subcommand == "remove" ||
			subcommand == "discover" || subcommand == "edit" {
			for _, chain := range w.Blockchains {
				if cmn.Contains(chain.Name, bchain) {
					options = append(options, ui.ACOption{
						Name:   chain.Name,
						Result: command + " " + subcommand + " " + strconv.Itoa(chain.ChainId) + " "})
				}
			}
			return "blockchain", &options, bchain

		}
	case 3:
		if subcommand == "add" {
			b := w.GetBlockchainByName(bchain)
			if b != nil {
				for _, lp := range cmn.PrefedinedLP_V3[b.ChainId] {
					options = append(options, ui.ACOption{
						Name:   lp.Name,
						Result: command + " " + subcommand + " '" + b.Name + "' '" + lp.Address.Hex() + "' '" + lp.Name + "' '" + lp.URL + "'"})
				}

				return "address", &options, addr
			}
		}

		if subcommand == "discover" || subcommand == "edit" {
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
			b := w.GetBlockchainByName(bchain)
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
			b := w.GetBlockchain(lp.ChainId)
			if b == nil {
				ui.PrintErrorf("LP_V3_Process:Blockchain not found: %d", lp.ChainId)
				w.RemoveLP_V3(lp.ChainId, lp.Provider)
				break
			}

			ui.Printf("%d %-12s %-12s ", i+1, b.Name, lp.Name)
			ui.Terminal.Screen.AddLink(gocui.ICON_EDIT, "command lp_v3 edit "+strconv.Itoa(lp.ChainId)+" '"+lp.Provider.Hex()+"' '"+lp.Name+"'", "Edit provider", "")
			ui.Terminal.Screen.AddLink(gocui.ICON_DELETE, "command lp_v3 remove "+strconv.Itoa(lp.ChainId)+" '"+lp.Provider.Hex()+"'", "Remove provider", "")
			cmn.AddAddressShortLink(ui.Terminal.Screen, lp.Provider)
			ui.Printf("\n")
		}

		ui.Printf("\n")

	case "add":
		b := w.GetBlockchainByName(chain)
		if b == nil {
			err = fmt.Errorf("LP_V3_Process: blockchain not found: %s", chain)
			break
		}

		bus.Send("ui", "popup", ui.DlgLP_V3_add(b, addr, name, url))
	case "edit":
		b := w.GetBlockchainByName(chain)
		if b == nil {
			err = fmt.Errorf("LP_V3_Process: blockchain not found: %s", chain)
			break
		}

		bus.Send("ui", "popup", ui.DlgLP_V3_edit(b, addr, name, url))
	case "remove":
		b := w.GetBlockchainByName(chain)
		if b == nil {
			err = fmt.Errorf("LP_V3_Process: blockchain not found: %s", chain)
			break
		}

		lp := w.GetLP_V3_by_name(b.ChainId, addr)
		if lp == nil {
			lp = w.GetLP_V3(b.ChainId, common.HexToAddress(addr))
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
			func() bool {
				err := w.RemoveLP_V3(b.ChainId, lp.Provider)
				if err != nil {
					ui.PrintErrorf("Error removing provider: %v", err)
					return false
				}
				ui.Notification.Show("Provider removed")
				return true
			}))

	case "discover":
		chain_id := 0
		b := w.GetBlockchainByName(chain)
		if b != nil {
			chain_id = b.ChainId
		}

		resp := bus.Fetch("lp_v3", "discover", bus.B_LP_V3_Discover{
			ChainId: chain_id,
			Name:    addr,
		})
		if resp.Error != nil {
			err = resp.Error
		}
	case "on":
		ui.LP_V3.ShowPane()
		w.LP_V3PaneOn = true
		err = w.Save()
	case "off":
		ui.LP_V3.HidePane()
		w.LP_V3PaneOn = false
		err = w.Save()
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

	ui.Printf("Xch@Chain     Pair    On Liq0     Liq1     Gain0    Gain1     Gain$    Fee%%    Address\n")

	for _, lp := range w.LP_V3_Positions {

		// sanity check
		if lp.Owner.Cmp(common.Address{}) == 0 {
			ui.PrintErrorf("No address, LP v3 position removed")
			w.RemoveLP_V3Position(lp.Owner, lp.ChainId, lp.Provider, lp.NFT_Token)
			log.Error().Msgf("No address, LP v3 position removed")
			continue
		}

		b := w.GetBlockchain(lp.ChainId)
		if b == nil {
			ui.PrintErrorf("Blockchain not found, V3 position removed: %d", lp.ChainId)
			w.RemoveLP_V3Position(lp.Owner, lp.ChainId, lp.Provider, lp.NFT_Token)
			log.Error().Msgf("Blockchain not found, V3 position removed: %d", lp.ChainId)
			continue
		}

		lpp := w.GetLP_V3(lp.ChainId, lp.Provider)
		if lpp == nil {
			ui.PrintErrorf("Provider not found, V3 position removed: %s", lp.Provider.String())
			w.RemoveLP_V3Position(lp.Owner, lp.ChainId, lp.Provider, lp.NFT_Token)
			log.Error().Msgf("Provider not found, V3 position removed: %s", lp.Provider.String())
			continue
		}

		a := w.GetAddress(lp.Owner)
		if a == nil {
			ui.PrintErrorf("Address not found, V3 position removed: %s", lp.Owner.String())
			w.RemoveLP_V3Position(lp.Owner, lp.ChainId, lp.Provider, lp.NFT_Token)
			log.Error().Msgf("Address not found, V3 position removed: %s", lp.Owner.String())
			continue
		}

		p_res := bus.Fetch("lp_v3", "get-position-status", &bus.B_LP_V3_GetPositionStatus{
			ChainId:   lp.ChainId,
			Provider:  lp.Provider,
			NFT_Token: lp.NFT_Token})
		if p_res.Error != nil {
			ui.PrintErrorf("Error fetching position status: %v", p_res.Error)
			continue
		}

		p, ok := p_res.Data.(*bus.B_LP_V3_GetPositionStatus_Response)
		if !ok {
			ui.PrintErrorf("Error fetching position status")
			continue
		}

		ui.Terminal.Screen.AddLink(
			fmt.Sprintf("%-12s ", p.ProviderName),
			"open "+lpp.URL,
			lpp.URL, "")

		t0 := w.GetTokenByAddress(p.ChainId, lp.Token0)
		t1 := w.GetTokenByAddress(p.ChainId, lp.Token1)

		if t0 != nil && t1 != nil {
			ui.Printf("%9s", t0.Symbol+"/"+t1.Symbol)
		} else {
			if t0 != nil {
				ui.Printf("%-5s", t0.Symbol)
			} else {
				ui.Terminal.Screen.AddLink("???", "command token add "+b.Name+" "+lp.Token0.String(), "Add token", "")
			}

			ui.Printf("/")

			if t1 != nil {
				ui.Printf("%-5s", t1.Symbol)
			} else {
				ui.Terminal.Screen.AddLink("???", "command token add "+b.Name+" "+lp.Token1.String(), "Add token", "")
			}
		}

		if p.On {
			ui.Printf(ui.F(gocui.ColorGreen) + gocui.ICON_LIGHT + ui.F(ui.Terminal.Screen.FgColor))
		} else {
			ui.Printf(ui.F(gocui.ColorRed) + gocui.ICON_LIGHT + ui.F(ui.Terminal.Screen.FgColor))
		}

		if t0 != nil {
			cmn.AddValueLink(ui.Terminal.Screen, p.Liquidity0, t0)
		} else {
			ui.Printf("                  ")
		}

		if t1 != nil {
			cmn.AddValueLink(ui.Terminal.Screen, p.Liquidity1, t1)
		} else {
			ui.Printf("                  ")
		}

		if t0 != nil {
			cmn.AddValueLink(ui.Terminal.Screen, p.Gain0, t0)
		} else {
			ui.Printf("                  ")
		}

		if t1 != nil {
			cmn.AddValueLink(ui.Terminal.Screen, p.Gain1, t1)
		} else {
			ui.Printf("                  ")
		}

		cmn.AddDollarLink(ui.Terminal.Screen, p.Dollars)

		// cmn.AddAddressShortLink(ui.Terminal.Screen, a.Address)

		ui.Printf("%2.1f/%2.1f ", p.FeeProtocol0, p.FeeProtocol1)
		ui.Printf(" %s\n", a.Name)
		ui.Flush()
	}

	ui.Printf("\n")
}
