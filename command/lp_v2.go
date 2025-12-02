package command

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/ethereum/go-ethereum/common"
)

var lp_v2_subcommands = []string{
	"on", "off", "add", "edit", "remove", "discover", "providers",
	"list", "set_api_key",
}

func NewLP_V2Command() *Command {
	return &Command{
		Command:      "lp_v2",
		ShortCommand: "v2",
		Usage: `
Usage: lp_v2 [COMMAND]

Manage v2 liquidity

Commands:
  list                      - List v2 positions
  providers                 - List v2 providers
  add [CHAIN] [PROVIDER]    - Add v2 provider
  remove [CHAIN] [NAME]     - Remove v2 provider
  edit [CHAIN] [NAME]       - Edit v2 provider
  discover [CHAIN] [NAME] [TOKEN0] [TOKEN1] - Discover v2 positions (optional token filters)
  set_api_key [KEY]         - Set The Graph API key
  on                        - Open v2 window
  off                       - Close v2 window
		`,
		Help:             `Manage liquidity v2`,
		Process:          LP_V2_Process,
		AutoCompleteFunc: LP_V2_AutoComplete,
	}
}

func LP_V2_AutoComplete(input string) (string, *[]ui.ACOption, string) {

	if cmn.CurrentWallet == nil {
		return "", nil, ""
	}

	w := cmn.CurrentWallet

	options := []ui.ACOption{}
	p := cmn.SplitN(input, 6)
	command, subcommand, bchain, addr, token0, token1 := p[0], p[1], p[2], p[3], p[4], p[5]

	last_param := len(p) - 1
	for last_param > 0 && p[last_param] == "" {
		last_param--
	}

	if strings.HasSuffix(input, " ") {
		last_param++
	}

	switch last_param {
	case 1:
		if !cmn.IsInArray(lp_v2_subcommands, subcommand) {
			for _, sc := range lp_v2_subcommands {
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
						Result: command + " " + subcommand + " '" + chain.Name + "' "})
				}
			}
			return "blockchain", &options, bchain

		}
	case 3:
		if subcommand == "add" {
			b := w.GetBlockchainByName(bchain)
			if b != nil {
				for _, lp := range cmn.PrefedinedLP_V2[b.ChainId] {
					options = append(options, ui.ACOption{
						Name:   lp.Name,
						Result: command + " " + subcommand + " '" + b.Name + "' '" + lp.Factory.Hex() + "' '" + lp.Router.Hex() + "' '" + lp.Name + "' '" + lp.URL + "' '" + lp.SubgraphID + "'"})
				}

				return "address", &options, addr
			}
		}

		if subcommand == "discover" || subcommand == "edit" {
			for _, lp := range w.LP_V2_Providers {
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
				for _, lp := range w.LP_V2_Providers {
					if lp.ChainId == b.ChainId && cmn.Contains(lp.Name, addr) {
						options = append(options, ui.ACOption{
							Name:   lp.Name,
							Result: command + " " + subcommand + " " + strconv.Itoa(lp.ChainId) + " '" + lp.Name + "'"})
					}
				}
			}
			return "address", &options, addr
		}
	case 4:
		// Token0 filter for discover command
		if subcommand == "discover" {
			b := w.GetBlockchainByName(bchain)
			if b == nil {
				// Try parsing as chain ID
				if chainId, err := strconv.Atoi(bchain); err == nil {
					b = w.GetBlockchain(chainId)
				}
			}
			if b != nil {
				for _, t := range w.Tokens {
					if t.ChainId == b.ChainId && cmn.Contains(t.Symbol, token0) {
						options = append(options, ui.ACOption{
							Name:   t.Symbol,
							Result: command + " " + subcommand + " '" + bchain + "' '" + addr + "' '" + t.Address.Hex() + "' "})
					}
				}
				return "token0", &options, token0
			}
		}
	case 5:
		// Token1 filter for discover command
		if subcommand == "discover" {
			b := w.GetBlockchainByName(bchain)
			if b == nil {
				// Try parsing as chain ID
				if chainId, err := strconv.Atoi(bchain); err == nil {
					b = w.GetBlockchain(chainId)
				}
			}
			if b != nil {
				for _, t := range w.Tokens {
					if t.ChainId == b.ChainId && cmn.Contains(t.Symbol, token1) {
						options = append(options, ui.ACOption{
							Name:   t.Symbol,
							Result: command + " " + subcommand + " '" + bchain + "' '" + addr + "' '" + token0 + "' '" + t.Address.Hex() + "'"})
					}
				}
				return "token1", &options, token1
			}
		}
	}
	return "", &options, ""
}

func LP_V2_Process(c *Command, input string) {
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

	p := cmn.SplitN(input, 10)
	_, subcommand, chain, factory, router, name, url, subgraphID := p[0], p[1], p[2], p[3], p[4], p[5], p[6], p[7]
	// For discover: p[4] = token0, p[5] = token1 (reusing router/name positions)

	switch subcommand {
	case "list", "":
		listV2(w)
	case "providers":
		ui.Printf("\nLP v2 Providers\n\n")

		if len(w.LP_V2_Providers) == 0 {
			ui.Printf("(no providers)\n")
		}

		sort.Slice(w.LP_V2_Providers, func(i, j int) bool {
			if w.LP_V2_Providers[i].ChainId == w.LP_V2_Providers[j].ChainId {
				return w.LP_V2_Providers[i].Name < w.LP_V2_Providers[j].Name
			}
			return w.LP_V2_Providers[i].ChainId < w.LP_V2_Providers[j].ChainId
		})

		for i, lp := range w.LP_V2_Providers {
			b := w.GetBlockchain(lp.ChainId)
			if b == nil {
				ui.PrintErrorf("LP_V2_Process:Blockchain not found: %d", lp.ChainId)
				w.RemoveLP_V2(lp.ChainId, lp.Factory)
				break
			}

			ui.Printf("%d %-12s %-12s ", i+1, b.Name, lp.Name)
			ui.Terminal.Screen.AddLink(cmn.ICON_EDIT, "command lp_v2 edit "+strconv.Itoa(lp.ChainId)+" '"+lp.Factory.Hex()+"' '"+lp.Name+"'", "Edit provider", "")
			ui.Terminal.Screen.AddLink(cmn.ICON_DELETE, "command lp_v2 remove "+strconv.Itoa(lp.ChainId)+" '"+lp.Factory.Hex()+"'", "Remove provider", "")
			cmn.AddAddressShortLink(ui.Terminal.Screen, lp.Factory)
			ui.Printf("\n")
		}

		ui.Printf("\n")

	case "add":
		b := w.GetBlockchainByName(chain)
		if b == nil {
			err = fmt.Errorf("LP_V2_Process: blockchain not found: %s", chain)
			break
		}

		bus.Send("ui", "popup", ui.DlgLP_V2_add(b, factory, router, name, url, subgraphID))
	case "edit":
		b := w.GetBlockchainByName(chain)
		if b == nil {
			err = fmt.Errorf("LP_V2_Process: blockchain not found: %s", chain)
			break
		}

		lp := w.GetLP_V2_by_name(b.ChainId, factory)
		if lp == nil {
			lp = w.GetLP_V2(b.ChainId, common.HexToAddress(factory))
		}
		if lp == nil {
			err = fmt.Errorf("provider not found: %s", factory)
			break
		}

		bus.Send("ui", "popup", ui.DlgLP_V2_edit(b, lp.Factory.Hex(), lp.Name, lp.URL, lp.SubgraphID))
	case "remove":
		b := w.GetBlockchainByName(chain)
		if b == nil {
			err = fmt.Errorf("LP_V2_Process: blockchain not found: %s", chain)
			break
		}

		lp := w.GetLP_V2_by_name(b.ChainId, factory)
		if lp == nil {
			lp = w.GetLP_V2(b.ChainId, common.HexToAddress(factory))
			if lp == nil {
				err = fmt.Errorf("provider not found: %s", factory)
				break
			}
		}

		bus.Send("ui", "popup", ui.DlgConfirm(
			"Remove provider",
			`
<c>Are you sure you want to remove provider?</c>

       Name:`+lp.Name+`
 Blockchain:`+b.Name+`
    Factory:`+lp.Factory.String()+`
`,
			func() bool {
				err := w.RemoveLP_V2(b.ChainId, lp.Factory)
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

		// For discover, token addresses are in positions 4 and 5 (router and name variables)
		var token0, token1 common.Address
		if router != "" {
			token0 = common.HexToAddress(router)
		}
		if name != "" {
			token1 = common.HexToAddress(name)
		}

		resp := bus.Fetch("lp_v2", "discover", bus.B_LP_V2_Discover{
			ChainId: chain_id,
			Name:    factory,
			Token0:  token0,
			Token1:  token1,
		})
		if resp.Error != nil {
			err = resp.Error
		}
	case "on":
		ui.ShowPane(&ui.LP_V2)
		w.LP_V2PaneOn = true
		err = w.Save()
	case "off":
		ui.HidePane(&ui.LP_V2)
		w.LP_V2PaneOn = false
		err = w.Save()
	case "set_api_key":
		apiKey := chain // the API key is in the 3rd position (chain param)
		if apiKey == "" {
			if cmn.Config.TheGraphAPIKey == "" {
				ui.Printf("The Graph API key is not set\n")
			} else {
				ui.Printf("The Graph API key: %s...%s\n", cmn.Config.TheGraphAPIKey[:8], cmn.Config.TheGraphAPIKey[len(cmn.Config.TheGraphAPIKey)-4:])
			}
		} else {
			cmn.Config.TheGraphAPIKey = apiKey
			cmn.ConfigChanged = true
			err = cmn.SaveConfig()
			if err == nil {
				ui.Printf("The Graph API key set successfully\n")
			}
		}
	default:
		err = fmt.Errorf("unknown command: %s", subcommand)
	}

	if err != nil {
		ui.PrintErrorf(err.Error())
	}

}

func listV2(w *cmn.Wallet) {
	ui.Printf("\nLP v2 Positions\n\n")

	if len(w.LP_V2_Positions) == 0 {
		ui.Printf("(no positions)\n")
		return
	}

	// Fetch position status for all positions
	list := make([]*bus.B_LP_V2_GetPositionStatus_Response, 0)
	for _, pos := range w.LP_V2_Positions {
		sr := bus.Fetch("lp_v2", "get-position-status", &bus.B_LP_V2_GetPositionStatus{
			ChainId: pos.ChainId,
			Factory: pos.Factory,
			Pair:    pos.Pair,
		})

		if sr.Error != nil {
			continue
		}

		resp, ok := sr.Data.(*bus.B_LP_V2_GetPositionStatus_Response)
		if !ok {
			continue
		}

		// Skip positions with ignored tokens
		t0 := w.GetTokenByAddress(resp.ChainId, resp.Token0)
		t1 := w.GetTokenByAddress(resp.ChainId, resp.Token1)
		if (t0 != nil && t0.Ignored) || (t1 != nil && t1.Ignored) {
			continue
		}

		list = append(list, resp)
	}

	// Sort by liquidity (descending)
	sort.Slice(list, func(i, j int) bool {
		if list[i].Liquidity0Dollars+list[i].Liquidity1Dollars == list[j].Liquidity0Dollars+list[j].Liquidity1Dollars {
			if list[i].ProviderName == list[j].ProviderName {
				return list[i].Owner.Hex() < list[j].Owner.Hex()
			}
			return list[i].ProviderName < list[j].ProviderName
		}
		return list[i].Liquidity0Dollars+list[i].Liquidity1Dollars > list[j].Liquidity0Dollars+list[j].Liquidity1Dollars
	})

	ui.Printf("Xch@Chain        Pair              Liq0      Liq1     Liq$ Address\n")

	for _, p := range list {
		provider := w.GetLP_V2(p.ChainId, p.Factory)
		if provider == nil {
			continue
		}

		b := w.GetBlockchain(p.ChainId)
		if b == nil {
			continue
		}

		owner := w.GetAddress(p.Owner)
		if owner == nil {
			continue
		}

		// ProviderName already includes @Chain from get_position_status
		ui.Terminal.Screen.AddLink(
			fmt.Sprintf("%-16s", p.ProviderName),
			"open "+provider.URL,
			"open "+provider.URL, "")

		t0 := w.GetTokenByAddress(p.ChainId, p.Token0)
		t1 := w.GetTokenByAddress(p.ChainId, p.Token1)

		var pairStr string
		var pairLen int
		if t0 != nil && t1 != nil {
			pairStr = t0.Symbol + "/" + t1.Symbol
			pairLen = len(pairStr)
			ui.Printf(" %s", pairStr)
		} else {
			ui.Printf(" ")
			if t0 != nil {
				ui.Printf("%s", t0.Symbol)
				pairLen = len(t0.Symbol)
			} else {
				ui.Terminal.Screen.AddLink("???", "command token add "+b.Name+" "+p.Token0.String(), "Add token", "")
				pairLen = 3
			}

			ui.Printf("/")
			pairLen++

			if t1 != nil {
				ui.Printf("%s", t1.Symbol)
				pairLen += len(t1.Symbol)
			} else {
				ui.Terminal.Screen.AddLink("???", "command token add "+b.Name+" "+p.Token1.String(), "Add token", "")
				pairLen += 3
			}
		}
		// Pad to 13 chars
		if pairLen < 13 {
			ui.Printf("%s", strings.Repeat(" ", 13-pairLen))
		}

		if t0 != nil {
			cmn.AddFixedValueLink(ui.Terminal.Screen, p.Liquidity0, t0, 10)
		} else {
			ui.Printf("          ")
		}

		if t1 != nil {
			cmn.AddFixedValueLink(ui.Terminal.Screen, p.Liquidity1, t1, 10)
		} else {
			ui.Printf("          ")
		}

		cmn.AddFixedDollarLink(ui.Terminal.Screen, p.Liquidity0Dollars+p.Liquidity1Dollars, 10)

		ui.Printf(" %s\n", owner.Name)
		ui.Flush()
	}

	ui.Printf("\n")
}
