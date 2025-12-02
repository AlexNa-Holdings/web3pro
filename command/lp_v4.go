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
)

var lp_v4_subcommands = []string{
	"on", "off", "add", "edit", "remove", "discover", "providers",
	"list", "set_api_key",
}

func NewLP_V4Command() *Command {
	return &Command{
		Command:      "lp_v4",
		ShortCommand: "v4",
		Usage: `
Usage: lp_v4 [COMMAND]

Manage v4 liquidity

Commands:
  list                      - List v4 positions
  providers                 - List v4 providers
  add [CHAIN] [PROVIDER]    - Add v4 provider
  remove [CHAIN] [NAME]     - Remove v4 provider
  edit [CHAIN] [NAME]       - Edit v4 provider
  discover [CHAIN] [NAME] [TOKEN_ID] - Discover v4 positions (optional token ID)
  set_api_key [KEY]         - Set The Graph API key
  on                        - Open v4 window
  off                       - Close v4 window
		`,
		Help:             `Manage liquidity v4`,
		Process:          LP_V4_Process,
		AutoCompleteFunc: LP_V4_AutoComplete,
	}
}

func LP_V4_AutoComplete(input string) (string, *[]ui.ACOption, string) {

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
		if !cmn.IsInArray(lp_v4_subcommands, subcommand) {
			for _, sc := range lp_v4_subcommands {
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
				for _, lp := range cmn.PrefedinedLP_V4[b.ChainId] {
					options = append(options, ui.ACOption{
						Name:   lp.Name,
						Result: command + " " + subcommand + " '" + b.Name + "' '" + lp.ProviderAddress.Hex() + "' '" + lp.PoolManager.Hex() + "' '" + lp.StateView.Hex() + "' '" + lp.Name + "' '" + lp.URL + "' '" + lp.SubgraphURL + "'"})
				}

				return "address", &options, addr
			}
		}

		if subcommand == "discover" || subcommand == "edit" {
			b := w.GetBlockchainByName(bchain)
			if b != nil {
				for _, lp := range w.LP_V4_Providers {
					if lp.ChainId == b.ChainId && cmn.Contains(lp.Name, addr) {
						options = append(options, ui.ACOption{
							Name:   lp.Name,
							Result: command + " " + subcommand + " '" + b.Name + "' '" + lp.Name + "'"})
					}
				}
			}
			return "name", &options, addr
		}

		if subcommand == "remove" {
			b := w.GetBlockchainByName(bchain)
			if b != nil {
				for _, lp := range w.LP_V4_Providers {
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

func LP_V4_Process(c *Command, input string) {
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

	p := cmn.SplitN(input, 9)
	_, subcommand, chain, provider, poolManager, stateView, name, url, subgraphURL := p[0], p[1], p[2], p[3], p[4], p[5], p[6], p[7], p[8]

	switch subcommand {
	case "list", "":
		listV4(w)
	case "providers":
		ui.Printf("\nLP v4 Providers\n\n")

		if len(w.LP_V4_Providers) == 0 {
			ui.Printf("(no providers)\n")
		}

		sort.Slice(w.LP_V4_Providers, func(i, j int) bool {
			if w.LP_V4_Providers[i].ChainId == w.LP_V4_Providers[j].ChainId {
				return w.LP_V4_Providers[i].Name < w.LP_V4_Providers[j].Name
			}
			return w.LP_V4_Providers[i].ChainId < w.LP_V4_Providers[j].ChainId
		})

		for i, lp := range w.LP_V4_Providers {
			b := w.GetBlockchain(lp.ChainId)
			if b == nil {
				ui.PrintErrorf("LP_V4_Process:Blockchain not found: %d", lp.ChainId)
				w.RemoveLP_V4(lp.ChainId, lp.Provider)
				break
			}

			ui.Printf("%d %-12s %-12s ", i+1, b.Name, lp.Name)
			ui.Terminal.Screen.AddLink(cmn.ICON_EDIT, "command lp_v4 edit "+strconv.Itoa(lp.ChainId)+" '"+lp.Provider.Hex()+"' '"+lp.Name+"'", "Edit provider", "")
			ui.Terminal.Screen.AddLink(cmn.ICON_DELETE, "command lp_v4 remove "+strconv.Itoa(lp.ChainId)+" '"+lp.Provider.Hex()+"'", "Remove provider", "")
			cmn.AddAddressShortLink(ui.Terminal.Screen, lp.Provider)
			ui.Printf("\n")
		}

		ui.Printf("\n")

	case "add":
		b := w.GetBlockchainByName(chain)
		if b == nil {
			err = fmt.Errorf("LP_V4_Process: blockchain not found: %s", chain)
			break
		}

		bus.Send("ui", "popup", ui.DlgLP_V4_add(b, provider, poolManager, stateView, name, url, subgraphURL))
	case "edit":
		b := w.GetBlockchainByName(chain)
		if b == nil {
			err = fmt.Errorf("LP_V4_Process: blockchain not found: %s", chain)
			break
		}

		lp := w.GetLP_V4_by_name(b.ChainId, provider)
		if lp == nil {
			lp = w.GetLP_V4(b.ChainId, common.HexToAddress(provider))
		}
		if lp == nil {
			err = fmt.Errorf("provider not found: %s", provider)
			break
		}

		bus.Send("ui", "popup", ui.DlgLP_V4_edit(b, lp.Provider.Hex(), lp.Name, lp.URL, lp.SubgraphURL))
	case "remove":
		b := w.GetBlockchainByName(chain)
		if b == nil {
			err = fmt.Errorf("LP_V4_Process: blockchain not found: %s", chain)
			break
		}

		lp := w.GetLP_V4_by_name(b.ChainId, provider)
		if lp == nil {
			lp = w.GetLP_V4(b.ChainId, common.HexToAddress(provider))
			if lp == nil {
				err = fmt.Errorf("provider not found: %s", provider)
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
				err := w.RemoveLP_V4(b.ChainId, lp.Provider)
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

		// For discover, token ID is in position 4 (poolManager variable)
		var tokenId *big.Int
		if poolManager != "" {
			tokenId, _ = new(big.Int).SetString(poolManager, 10)
		}

		resp := bus.Fetch("lp_v4", "discover", bus.B_LP_V4_Discover{
			ChainId: chain_id,
			Name:    provider,
			TokenId: tokenId,
		})
		if resp.Error != nil {
			err = resp.Error
		}
	case "on":
		ui.ShowPane(&ui.LP_V4)
		w.LP_V4PaneOn = true
		err = w.Save()
	case "off":
		ui.HidePane(&ui.LP_V4)
		w.LP_V4PaneOn = false
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

func listV4(w *cmn.Wallet) {
	ui.Printf("\nLP v4 Positions\n\n")

	if len(w.LP_V4_Positions) == 0 {
		ui.Printf("(no positions)\n")
		return
	}

	// Fetch position status for all positions
	list := make([]*bus.B_LP_V4_GetPositionStatus_Response, 0)
	for _, pos := range w.LP_V4_Positions {
		sr := bus.Fetch("lp_v4", "get-position-status", &bus.B_LP_V4_GetPositionStatus{
			ChainId:   pos.ChainId,
			Provider:  pos.Provider,
			NFT_Token: pos.NFT_Token,
		})

		if sr.Error != nil {
			continue
		}

		resp, ok := sr.Data.(*bus.B_LP_V4_GetPositionStatus_Response)
		if !ok {
			continue
		}

		// Skip positions with ignored tokens
		t0 := w.GetTokenByAddress(resp.ChainId, resp.Currency0)
		t1 := w.GetTokenByAddress(resp.ChainId, resp.Currency1)
		if (t0 != nil && t0.Ignored) || (t1 != nil && t1.Ignored) {
			continue
		}

		list = append(list, resp)
	}

	// Sort by gain (descending)
	sort.Slice(list, func(i, j int) bool {
		if list[i].Gain0Dollars+list[i].Gain1Dollars == list[j].Gain0Dollars+list[j].Gain1Dollars {
			if list[i].ProviderName == list[j].ProviderName {
				return list[i].Owner.Hex() < list[j].Owner.Hex()
			}
			return list[i].ProviderName < list[j].ProviderName
		}
		return list[i].Gain0Dollars+list[i].Gain1Dollars > list[j].Gain0Dollars+list[j].Gain1Dollars
	})

	ui.Printf("Xch@Chain        Pair   On Liq0     Liq1     Gain0    Gain1     Gain$  Address\n")

	for _, p := range list {
		provider := w.GetLP_V4(p.ChainId, p.Provider)
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

		ui.Terminal.Screen.AddLink(
			fmt.Sprintf("%-12s", p.ProviderName),
			"open "+provider.URL,
			"open "+provider.URL, "")

		t0 := w.GetTokenByAddress(p.ChainId, p.Currency0)
		t1 := w.GetTokenByAddress(p.ChainId, p.Currency1)

		if t0 != nil && t1 != nil {
			ui.Printf("%11s", t0.Symbol+"/"+t1.Symbol)
		} else {
			if t0 != nil {
				ui.Printf("%-5s", t0.Symbol)
			} else {
				ui.Terminal.Screen.AddLink("???", "command token add "+b.Name+" "+p.Currency0.String(), "Add token", "")
			}

			ui.Printf("/")

			if t1 != nil {
				ui.Printf("%-5s", t1.Symbol)
			} else {
				ui.Terminal.Screen.AddLink("???", "command token add "+b.Name+" "+p.Currency1.String(), "Add token", "")
			}
		}

		if p.On {
			ui.Printf(ui.F(gocui.ColorGreen) + cmn.ICON_LIGHT + ui.F(ui.Terminal.Screen.FgColor))
		} else {
			ui.Printf(ui.F(gocui.ColorRed) + cmn.ICON_LIGHT + ui.F(ui.Terminal.Screen.FgColor))
		}

		if t0 != nil {
			cmn.AddValueLink(ui.Terminal.Screen, p.Liquidity0, t0)
		} else {
			ui.Printf("         ")
		}

		if t1 != nil {
			cmn.AddValueLink(ui.Terminal.Screen, p.Liquidity1, t1)
		} else {
			ui.Printf("         ")
		}

		if t0 != nil {
			cmn.AddValueLink(ui.Terminal.Screen, p.Gain0, t0)
		} else {
			ui.Printf("         ")
		}

		if t1 != nil {
			cmn.AddValueLink(ui.Terminal.Screen, p.Gain1, t1)
		} else {
			ui.Printf("         ")
		}

		cmn.AddDollarLink(ui.Terminal.Screen, p.Gain0Dollars+p.Gain1Dollars)

		ui.Printf(" %s\n", owner.Name)
		ui.Flush()
	}

	ui.Printf("\n")
}
