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

var token_subcommands = []string{"on", "off", "remove", "edit", "add", "balance", "list", "ignore", "unignore", "ignored"}

func NewTokenCommand() *Command {
	return &Command{
		Command:      "token",
		ShortCommand: "t",
		Subcommands:  token_subcommands,
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
  ignore [BLOCKCHAIN] [TOKEN]   - Ignore token (hide from LP positions)
  unignore [BLOCKCHAIN] [TOKEN] - Unignore token
  ignored                       - List ignored tokens
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
		if subcommand == "list" || subcommand == "add" || subcommand == "remove" || subcommand == "balance" || subcommand == "edit" || subcommand == "ignore" || subcommand == "unignore" {
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

		if subcommand == "ignore" && b != nil {
			for _, t := range w.Tokens {
				if t.ChainId != b.ChainId || t.Ignored {
					continue
				}
				if cmn.Contains(t.Symbol, token) || cmn.Contains(t.Address.String(), token) || cmn.Contains(t.Name, token) {
					options = append(options, ui.ACOption{
						Name:   fmt.Sprintf("%-6s %s", t.Symbol, t.GetPrintName()),
						Result: fmt.Sprintf("%s ignore %d '%s'", command, b.ChainId, t.Address.String())})
				}
			}
			return "token", &options, token
		}

		if subcommand == "unignore" && b != nil {
			for _, t := range w.Tokens {
				if t.ChainId != b.ChainId || !t.Ignored {
					continue
				}
				if cmn.Contains(t.Symbol, token) || cmn.Contains(t.Address.String(), token) || cmn.Contains(t.Name, token) {
					options = append(options, ui.ACOption{
						Name:   fmt.Sprintf("%-6s %s", t.Symbol, t.GetPrintName()),
						Result: fmt.Sprintf("%s unignore %d '%s'", command, b.ChainId, t.Address.String())})
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

		if symbol == "" {
			ui.PrintErrorf("Token has empty symbol - contract may not be a valid ERC20 token")
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

			// Skip ignored tokens (use 'token ignored' to see them)
			if t.Ignored {
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

		// If no params, show token balance table similar to tokens pane
		if chain == "" {
			showTokenBalanceTable(w)
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

	case "ignore":
		chain := p[2]
		token := p[3]

		b := w.GetBlockchainByName(chain)
		if b == nil {
			chain_id, err := strconv.Atoi(chain)
			if err != nil {
				ui.PrintErrorf("Invalid blockchain: %s", chain)
				return
			}
			b = w.GetBlockchain(chain_id)
			if b == nil {
				ui.PrintErrorf("Blockchain not found: %s", chain)
				return
			}
		}

		if token == "" {
			ui.PrintErrorf("Usage: token ignore [BLOCKCHAIN] [TOKEN/ADDRESS]")
			return
		}

		t := w.GetTokenBySymbol(b.ChainId, token)
		if t == nil {
			t = w.GetTokenByAddress(b.ChainId, common.HexToAddress(token))
		}

		if t == nil {
			ui.PrintErrorf("Token not found: %s", token)
			return
		}

		t.Ignored = true
		if err := w.Save(); err != nil {
			ui.PrintErrorf("Error saving wallet: %v", err)
			return
		}
		ui.Printf("Token %s ignored\n", t.Symbol)

	case "unignore":
		chain := p[2]
		token := p[3]

		b := w.GetBlockchainByName(chain)
		if b == nil {
			chain_id, err := strconv.Atoi(chain)
			if err != nil {
				ui.PrintErrorf("Invalid blockchain: %s", chain)
				return
			}
			b = w.GetBlockchain(chain_id)
			if b == nil {
				ui.PrintErrorf("Blockchain not found: %s", chain)
				return
			}
		}

		if token == "" {
			ui.PrintErrorf("Usage: token unignore [BLOCKCHAIN] [TOKEN/ADDRESS]")
			return
		}

		t := w.GetTokenBySymbol(b.ChainId, token)
		if t == nil {
			t = w.GetTokenByAddress(b.ChainId, common.HexToAddress(token))
		}

		if t == nil {
			ui.PrintErrorf("Token not found: %s", token)
			return
		}

		t.Ignored = false
		if err := w.Save(); err != nil {
			ui.PrintErrorf("Error saving wallet: %v", err)
			return
		}
		ui.Printf("Token %s unignored\n", t.Symbol)

	case "ignored":
		ui.Printf("\nIgnored Tokens:\n\n")

		hasIgnored := false
		for _, t := range w.Tokens {
			if !t.Ignored {
				continue
			}
			hasIgnored = true

			b := w.GetBlockchain(t.ChainId)
			if b == nil {
				continue
			}

			ui.Printf("%-8s ", t.Symbol)
			ui.Terminal.Screen.AddLink(cmn.ICON_CHECK, fmt.Sprintf("command token unignore %d '%s'", t.ChainId, t.Address.String()), "Unignore token", "")

			if !t.Native {
				cmn.AddFixedAddressShortLink(ui.Terminal.Screen, t.Address, 15)
			} else {
				ui.Printf("%-15s", "Native")
			}

			ui.Printf(" %s\n", b.Name)
		}

		if !hasIgnored {
			ui.Printf("(no ignored tokens)\n")
		}
		ui.Printf("\n")

	default:
		ui.PrintErrorf("Invalid subcommand: %s", subcommand)
	}

}

// showTokenBalanceTable displays a token balance table similar to the tokens pane
func showTokenBalanceTable(w *cmn.Wallet) {
	type tokenInfo struct {
		Token         *cmn.Token
		LiquidBalance *big.Int
		StakedBalance *big.Int
		LPBalance     *big.Int
		TotalBalance  *big.Int
		TotalUSD      float64
	}

	ui.Printf("\nLoading balances...\n")

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

		// Handle native tokens with batch query
		if len(nativeTokens) > 0 && len(w.Addresses) > 0 {
			queries := make([]*eth.NativeBalanceQuery, 0, len(w.Addresses))
			for _, addr := range w.Addresses {
				queries = append(queries, &eth.NativeBalanceQuery{
					Holder: addr.Address,
				})
			}

			err := eth.GetNativeBalancesBatch(b, queries)
			if err != nil {
				log.Debug().Err(err).Msgf("Batch native balance query failed for chain %d", chainId)
			}

			totalBalance := big.NewInt(0)
			for _, q := range queries {
				if q.Balance != nil {
					totalBalance.Add(totalBalance, q.Balance)
				}
			}

			for _, t := range nativeTokens {
				tokenBalances[t] = new(big.Int).Set(totalBalance)
			}
		}

		// Handle ERC20 tokens with batch query
		if len(erc20Tokens) > 0 && len(w.Addresses) > 0 {
			queries := make([]*eth.BalanceQuery, 0, len(erc20Tokens)*len(w.Addresses))
			queryMap := make(map[*eth.BalanceQuery]*cmn.Token)

			for _, t := range erc20Tokens {
				for _, addr := range w.Addresses {
					q := &eth.BalanceQuery{
						Token:  t,
						Holder: addr.Address,
					}
					queries = append(queries, q)
					queryMap[q] = t
				}
			}

			err := eth.GetERC20BalancesBatch(b, queries)
			if err != nil {
				log.Debug().Err(err).Msgf("Batch balance query failed for chain %d", chainId)
			}

			for _, q := range queries {
				t := queryMap[q]
				if tokenBalances[t] == nil {
					tokenBalances[t] = big.NewInt(0)
				}
				if q.Balance != nil {
					tokenBalances[t].Add(tokenBalances[t], q.Balance)
				}
			}
		}
	}

	// Calculate staked balances from staking positions
	stakedBalances := make(map[*cmn.Token]*big.Int)
	for _, pos := range w.StakingPositions {
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

	// Calculate LP balances from LP positions (V2, V3, V4)
	lpBalances := make(map[*cmn.Token]*big.Int)

	// LP V2 positions
	for _, pos := range w.LP_V2_Positions {
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
