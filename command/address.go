package command

import (
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

var address_subcommands = []string{"remove", "edit", "list", "set", "add", "balances"}

func NewAddressCommand() *Command {
	return &Command{
		Command:      "address",
		ShortCommand: "a",
		Subcommands:  address_subcommands,
		Usage: `
Usage: address [COMMAND]

Manage addresses

Commands:
  add [ADDRESS]      - Add watch-only address
  set [ADDRESS]      - Set the current address
  list               - List addresses
  edit [ADDRESS]     - Edit address
  remove [ADDRESS]   - Remove address
  balances [ADDRESS] - Show token balances for address

Note: To add addresses with a signer, use 'signer addresses' command.
		`,
		Help:             `Manage addresses`,
		Process:          Address_Process,
		AutoCompleteFunc: Address_AutoComplete,
	}
}

func Address_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}

	if cmn.CurrentWallet == nil {
		return "", &options, ""
	}

	w := cmn.CurrentWallet

	p := cmn.SplitN(input, 5)
	command, subcommand, param := p[0], p[1], p[2]

	if !cmn.IsInArray(address_subcommands, subcommand) {
		for _, sc := range address_subcommands {
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	}

	if subcommand == "set" || subcommand == "remove" || subcommand == "edit" || subcommand == "balances" {
		for _, a := range w.Addresses {
			if cmn.Contains(a.Name+a.Address.String(), param) {
				options = append(options, ui.ACOption{
					Name:   cmn.ShortAddress(a.Address) + " " + a.Name,
					Result: command + " " + subcommand + " '" + a.Name + "'"})
			}
		}
		return "address", &options, param
	}

	return "", &options, ""
}

func Address_Process(c *Command, input string) {
	if cmn.CurrentWallet == nil {
		ui.PrintErrorf("No wallet open")
		return
	}

	w := cmn.CurrentWallet

	//parse command subcommand parameters
	tokens := cmn.SplitN(input, 3)
	_, subcommand, p0 := tokens[0], tokens[1], tokens[2]

	switch subcommand {
	case "add":
		if !common.IsHexAddress(p0) {
			ui.PrintErrorf("Invalid address")
			return
		}
		bus.Send("ui", "popup", ui.DlgAddressAddWatch(p0))
	case "remove":
		for i, a := range w.Addresses {
			if a.Name == p0 {
				bus.Send("ui", "popup", ui.DlgConfirm(
					"Remove address",
					`
<c>Are you sure you want to remove address:
<c> `+a.Name+`
<c> `+a.Address.String()+"? \n",
					func() bool {
						w.Addresses = append(w.Addresses[:i], w.Addresses[i+1:]...)

						err := w.Save()
						if err != nil {
							ui.PrintErrorf("Error saving wallet: %v", err)
							return false
						}
						ui.Notification.Show("Address removed")
						return true
					}))

				return
			}
		}
		ui.PrintErrorf("Address not found: %s", p0)
	case "list", "":

		sort.Slice(w.Addresses, func(i, j int) bool {
			return w.Addresses[i].Name < w.Addresses[j].Name
		})
		ui.Printf("\nAddresses:\n")
		for _, a := range w.Addresses {
			cmn.AddAddressShortLink(ui.Terminal.Screen, a.Address)
			ui.Printf(" ")
			ui.Terminal.Screen.AddLink(cmn.ICON_EDIT, "command address edit '"+a.Name+"'", "Edit address", "")
			ui.Terminal.Screen.AddLink(cmn.ICON_DELETE, "command address remove '"+a.Name+"'", "Remove address", "")
			signerInfo := a.Signer
			if signerInfo == "" {
				signerInfo = "watch"
			}
			ui.Printf(" %-14s (%s) \n", a.Name, signerInfo)
		}

	case "edit":
		if w.GetAddressByName(p0) == nil {
			ui.PrintErrorf("Address not found: %s", p0)
			return
		}
		bus.Send("ui", "popup", ui.DlgAddressEdit(p0))
	case "set":
		fa := w.GetAddressByName(p0)
		if fa == nil {
			ui.PrintErrorf("Address not found: %s", p0)
			return
		}
		w.CurrentAddress = fa.Address
		err := w.Save()
		if err != nil {
			ui.PrintErrorf("Error saving wallet: %v", err)
			return
		}

	case "balances":
		if p0 == "" {
			// Show address selection helper
			ui.Printf("\nSelect address:\n")
			for _, a := range w.Addresses {
				cmn.AddAddressShortLink(ui.Terminal.Screen, a.Address)
				ui.Printf(" ")
				ui.Terminal.Screen.AddLink(a.Name, "command address balances '"+a.Name+"'", "Show balances for "+a.Name, "")
				ui.Printf("\n")
			}
			return
		}
		a := w.GetAddressByName(p0)
		if a == nil {
			ui.PrintErrorf("Address not found: %s", p0)
			return
		}
		showAddressBalances(w, a)

	default:
		ui.PrintErrorf("Invalid subcommand: %s", subcommand)
	}

}

// showAddressBalances displays token balances for a specific address
func showAddressBalances(w *cmn.Wallet, addr *cmn.Address) {
	type tokenInfo struct {
		Token         *cmn.Token
		LiquidBalance *big.Int
		StakedBalance *big.Int
		LPBalance     *big.Int
		TotalBalance  *big.Int
		TotalUSD      float64
	}

	ui.Printf("\nLoading balances for %s (%s)...\n", addr.Name, cmn.ShortAddress(addr.Address))

	// Group tokens by chain for batch processing
	tokensByChain := make(map[int][]*cmn.Token)
	for _, t := range w.Tokens {
		if t.Ignored {
			continue
		}
		tokensByChain[t.ChainId] = append(tokensByChain[t.ChainId], t)
	}

	// Collect liquid balances using batch queries
	tokenBalances := make(map[*cmn.Token]*big.Int)

	for chainId, tokens := range tokensByChain {
		b := w.GetBlockchain(chainId)
		if b == nil {
			continue
		}

		// Separate native and ERC20 tokens
		var nativeTokens []*cmn.Token
		var erc20Tokens []*cmn.Token
		for _, t := range tokens {
			if t.Native {
				nativeTokens = append(nativeTokens, t)
			} else {
				erc20Tokens = append(erc20Tokens, t)
			}
		}

		// Handle native tokens
		if len(nativeTokens) > 0 {
			queries := []*eth.NativeBalanceQuery{{Holder: addr.Address}}

			err := eth.GetNativeBalancesBatch(b, queries)
			if err != nil {
				log.Debug().Err(err).Msgf("Native balance query failed for chain %d", chainId)
			}

			if queries[0].Balance != nil {
				for _, t := range nativeTokens {
					tokenBalances[t] = new(big.Int).Set(queries[0].Balance)
				}
			}
		}

		// Handle ERC20 tokens with batch query
		if len(erc20Tokens) > 0 {
			queries := make([]*eth.BalanceQuery, 0, len(erc20Tokens))
			queryMap := make(map[*eth.BalanceQuery]*cmn.Token)

			for _, t := range erc20Tokens {
				q := &eth.BalanceQuery{
					Token:  t,
					Holder: addr.Address,
				}
				queries = append(queries, q)
				queryMap[q] = t
			}

			err := eth.GetERC20BalancesBatch(b, queries)
			if err != nil {
				log.Debug().Err(err).Msgf("Batch balance query failed for chain %d", chainId)
			}

			for _, q := range queries {
				t := queryMap[q]
				if q.Balance != nil && q.Balance.Sign() > 0 {
					tokenBalances[t] = q.Balance
				}
			}
		}
	}

	// Calculate staked balances from staking positions for this address
	stakedBalances := make(map[*cmn.Token]*big.Int)
	for _, pos := range w.StakingPositions {
		if pos.Owner != addr.Address {
			continue
		}

		s := w.GetStaking(pos.ChainId, pos.Contract)
		if s == nil {
			continue
		}

		stakedToken := w.GetTokenByAddress(s.ChainId, s.StakedToken)
		if stakedToken == nil {
			continue
		}

		balResp := bus.Fetch("staking", "get-balance", &bus.B_Staking_GetBalance{
			ChainId:      s.ChainId,
			Contract:     s.Contract,
			Owner:        pos.Owner,
			ValidatorId:  pos.ValidatorId,
			VaultAddress: pos.VaultAddress,
		})

		if balResp.Error == nil {
			if balance, ok := balResp.Data.(*bus.B_Staking_GetBalance_Response); ok && balance.Balance != nil {
				if stakedBalances[stakedToken] == nil {
					stakedBalances[stakedToken] = big.NewInt(0)
				}
				stakedBalances[stakedToken].Add(stakedBalances[stakedToken], balance.Balance)
			}
		}
	}

	// Add pending staking rewards to staked balances
	for _, pos := range w.StakingPositions {
		if pos.Owner != addr.Address {
			continue
		}

		s := w.GetStaking(pos.ChainId, pos.Contract)
		if s == nil {
			continue
		}

		// Reward 1
		if s.Reward1Func != "" || s.Hardcoded {
			rewardToken := w.GetTokenByAddress(s.ChainId, s.Reward1Token)
			if rewardToken == nil && s.Reward1Token == ([20]byte{}) {
				b := w.GetBlockchain(s.ChainId)
				if b != nil {
					rewardToken, _ = w.GetNativeToken(b)
				}
			}
			if rewardToken != nil {
				pendingResp := bus.Fetch("staking", "get-pending", &bus.B_Staking_GetPending{
					ChainId:      s.ChainId,
					Contract:     s.Contract,
					Owner:        pos.Owner,
					RewardToken:  s.Reward1Token,
					ValidatorId:  pos.ValidatorId,
					VaultAddress: pos.VaultAddress,
				})
				if pendingResp.Error == nil {
					if pending, ok := pendingResp.Data.(*bus.B_Staking_GetPending_Response); ok && pending.Pending != nil && pending.Pending.Sign() > 0 {
						if stakedBalances[rewardToken] == nil {
							stakedBalances[rewardToken] = big.NewInt(0)
						}
						stakedBalances[rewardToken].Add(stakedBalances[rewardToken], pending.Pending)
					}
				}
			}
		}

		// Reward 2
		if s.Reward2Token != ([20]byte{}) && s.Reward2Func != "" {
			rewardToken := w.GetTokenByAddress(s.ChainId, s.Reward2Token)
			if rewardToken != nil {
				pendingResp := bus.Fetch("staking", "get-pending", &bus.B_Staking_GetPending{
					ChainId:      s.ChainId,
					Contract:     s.Contract,
					Owner:        pos.Owner,
					RewardToken:  s.Reward2Token,
					ValidatorId:  pos.ValidatorId,
					VaultAddress: pos.VaultAddress,
				})
				if pendingResp.Error == nil {
					if pending, ok := pendingResp.Data.(*bus.B_Staking_GetPending_Response); ok && pending.Pending != nil && pending.Pending.Sign() > 0 {
						if stakedBalances[rewardToken] == nil {
							stakedBalances[rewardToken] = big.NewInt(0)
						}
						stakedBalances[rewardToken].Add(stakedBalances[rewardToken], pending.Pending)
					}
				}
			}
		}
	}

	// Calculate LP balances from LP positions for this address
	lpBalances := make(map[*cmn.Token]*big.Int)

	// LP V2 positions
	for _, pos := range w.LP_V2_Positions {
		if pos.Owner != addr.Address {
			continue
		}
		sr := bus.Fetch("lp_v2", "get-position-status", &bus.B_LP_V2_GetPositionStatus{
			ChainId: pos.ChainId,
			Factory: pos.Factory,
			Pair:    pos.Pair,
		})
		if sr.Error == nil {
			if resp, ok := sr.Data.(*bus.B_LP_V2_GetPositionStatus_Response); ok {
				if t0 := w.GetTokenByAddress(resp.ChainId, resp.Token0); t0 != nil && resp.Liquidity0 != nil {
					if lpBalances[t0] == nil {
						lpBalances[t0] = big.NewInt(0)
					}
					lpBalances[t0].Add(lpBalances[t0], resp.Liquidity0)
				}
				if t1 := w.GetTokenByAddress(resp.ChainId, resp.Token1); t1 != nil && resp.Liquidity1 != nil {
					if lpBalances[t1] == nil {
						lpBalances[t1] = big.NewInt(0)
					}
					lpBalances[t1].Add(lpBalances[t1], resp.Liquidity1)
				}
			}
		}
	}

	// LP V3 positions
	for _, pos := range w.LP_V3_Positions {
		if pos.Owner != addr.Address {
			continue
		}
		sr := bus.Fetch("lp_v3", "get-position-status", &bus.B_LP_V3_GetPositionStatus{
			ChainId:   pos.ChainId,
			Provider:  pos.Provider,
			NFT_Token: pos.NFT_Token,
		})
		if sr.Error == nil {
			if resp, ok := sr.Data.(*bus.B_LP_V3_GetPositionStatus_Response); ok {
				if t0 := w.GetTokenByAddress(resp.ChainId, resp.Token0); t0 != nil {
					if lpBalances[t0] == nil {
						lpBalances[t0] = big.NewInt(0)
					}
					if resp.Liquidity0 != nil {
						lpBalances[t0].Add(lpBalances[t0], resp.Liquidity0)
					}
					if resp.Gain0 != nil {
						lpBalances[t0].Add(lpBalances[t0], resp.Gain0)
					}
				}
				if t1 := w.GetTokenByAddress(resp.ChainId, resp.Token1); t1 != nil {
					if lpBalances[t1] == nil {
						lpBalances[t1] = big.NewInt(0)
					}
					if resp.Liquidity1 != nil {
						lpBalances[t1].Add(lpBalances[t1], resp.Liquidity1)
					}
					if resp.Gain1 != nil {
						lpBalances[t1].Add(lpBalances[t1], resp.Gain1)
					}
				}
			}
		}
	}

	// LP V4 positions
	for _, pos := range w.LP_V4_Positions {
		if pos.Owner != addr.Address {
			continue
		}
		sr := bus.Fetch("lp_v4", "get-position-status", &bus.B_LP_V4_GetPositionStatus{
			ChainId:   pos.ChainId,
			Provider:  pos.Provider,
			NFT_Token: pos.NFT_Token,
		})
		if sr.Error == nil {
			if resp, ok := sr.Data.(*bus.B_LP_V4_GetPositionStatus_Response); ok {
				if t0 := w.GetTokenByAddress(resp.ChainId, resp.Currency0); t0 != nil {
					if lpBalances[t0] == nil {
						lpBalances[t0] = big.NewInt(0)
					}
					if resp.Liquidity0 != nil {
						lpBalances[t0].Add(lpBalances[t0], resp.Liquidity0)
					}
					if resp.Gain0 != nil {
						lpBalances[t0].Add(lpBalances[t0], resp.Gain0)
					}
				}
				if t1 := w.GetTokenByAddress(resp.ChainId, resp.Currency1); t1 != nil {
					if lpBalances[t1] == nil {
						lpBalances[t1] = big.NewInt(0)
					}
					if resp.Liquidity1 != nil {
						lpBalances[t1].Add(lpBalances[t1], resp.Liquidity1)
					}
					if resp.Gain1 != nil {
						lpBalances[t1].Add(lpBalances[t1], resp.Gain1)
					}
				}
			}
		}
	}

	// Build the list
	allTokens := make(map[*cmn.Token]bool)
	for t := range tokenBalances {
		allTokens[t] = true
	}
	for t := range stakedBalances {
		allTokens[t] = true
	}
	for t := range lpBalances {
		allTokens[t] = true
	}

	list := make([]*tokenInfo, 0)
	totalUSD := 0.0

	for t := range allTokens {
		liquidBalance := tokenBalances[t]
		if liquidBalance == nil {
			liquidBalance = big.NewInt(0)
		}

		stakedBalance := stakedBalances[t]
		if stakedBalance == nil {
			stakedBalance = big.NewInt(0)
		}

		lpBalance := lpBalances[t]
		if lpBalance == nil {
			lpBalance = big.NewInt(0)
		}

		totalBalance := new(big.Int).Add(liquidBalance, stakedBalance)
		totalBalance.Add(totalBalance, lpBalance)

		if totalBalance.Cmp(big.NewInt(0)) == 0 {
			continue
		}

		usdValue := t.Price * t.Float64(totalBalance)

		if usdValue < cmn.Config.MinTokenValue {
			continue
		}

		totalUSD += usdValue

		list = append(list, &tokenInfo{
			Token:         t,
			LiquidBalance: liquidBalance,
			StakedBalance: stakedBalance,
			LPBalance:     lpBalance,
			TotalBalance:  totalBalance,
			TotalUSD:      usdValue,
		})
	}

	// Sort by USD value descending
	sort.Slice(list, func(i, j int) bool {
		return list[i].TotalUSD > list[j].TotalUSD
	})

	if len(list) == 0 {
		ui.Printf("\n(no token balances found)\n")
		return
	}

	// Print table header
	ui.Printf("\nSymbol   Chain Price   Change   Liquid    Staked    LP        Total         Value\n")

	for _, ti := range list {
		t := ti.Token

		b := w.GetBlockchain(t.ChainId)
		if b == nil {
			continue
		}

		ui.Printf("%-8s ", t.Symbol)
		ui.Printf("%-5s ", b.GetShortName())

		// Price with clickable link
		if t.Price > 0 {
			cmn.AddFixedDollarLink(ui.Terminal.Screen, t.Price, 10)
		} else {
			ui.Printf("          ")
		}

		// Price change with color
		if t.PriceChange24 > 0 {
			ui.Printf(ui.F(gocui.ColorGreen)+"\uf0d8%5.2f%% "+ui.F(ui.Terminal.Screen.FgColor), t.PriceChange24)
		} else if t.PriceChange24 < 0 {
			ui.Printf(ui.F(gocui.ColorRed)+"\uf0d7%5.2f%% "+ui.F(ui.Terminal.Screen.FgColor), -t.PriceChange24)
		} else {
			ui.Printf("         ")
		}

		// Liquid balance with clickable link
		if ti.LiquidBalance != nil && ti.LiquidBalance.Sign() > 0 {
			cmn.AddValueLink(ui.Terminal.Screen, ti.LiquidBalance, t)
		} else {
			ui.Printf("         ")
		}

		// Staked balance with clickable link
		if ti.StakedBalance != nil && ti.StakedBalance.Sign() > 0 {
			cmn.AddValueLink(ui.Terminal.Screen, ti.StakedBalance, t)
		} else {
			ui.Printf("         ")
		}

		// LP balance with clickable link
		if ti.LPBalance != nil && ti.LPBalance.Sign() > 0 {
			cmn.AddValueLink(ui.Terminal.Screen, ti.LPBalance, t)
		} else {
			ui.Printf("         ")
		}

		// Total balance with clickable link
		cmn.AddValueLink(ui.Terminal.Screen, ti.TotalBalance, t)

		// USD value with clickable link
		cmn.AddDollarLink(ui.Terminal.Screen, ti.TotalUSD)

		ui.Printf("\n")
	}

	ui.Printf("\nTotal: ")
	cmn.AddDollarLink(ui.Terminal.Screen, totalUSD)
	ui.Printf("\n")
}
