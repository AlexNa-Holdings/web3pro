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
	"github.com/rs/zerolog/log"
)

var staking_subcommands = []string{
	"list", "providers", "add", "edit", "remove", "discover", "on", "off",
}

func NewStakingCommand() *Command {
	return &Command{
		Command:      "staking",
		ShortCommand: "st",
		Usage: `
Usage: staking [COMMAND]

Manage staking contracts

Commands:
  list                        - List staking positions
  providers                   - List staking providers (contracts)
  add [NAME]                  - Add staking provider
  edit [CHAIN] [NAME]         - Edit staking provider
  remove [CHAIN] [NAME]       - Remove staking provider
  discover [CHAIN] [PROVIDER] [ADDRESS] - Discover staking positions (optional filters)
  on                          - Show staking pane
  off                         - Hide staking pane
		`,
		Help:             `Manage staking contracts`,
		Process:          Staking_Process,
		AutoCompleteFunc: Staking_AutoComplete,
	}
}

func Staking_AutoComplete(input string) (string, *[]ui.ACOption, string) {

	if cmn.CurrentWallet == nil {
		return "", nil, ""
	}

	w := cmn.CurrentWallet

	options := []ui.ACOption{}
	p := cmn.SplitN(input, 6)
	command, subcommand, bchain, name, addrStr := p[0], p[1], p[2], p[3], p[4]

	last_param := len(p) - 1
	for last_param > 0 && p[last_param] == "" {
		last_param--
	}

	if strings.HasSuffix(input, " ") {
		last_param++
	}

	switch last_param {
	case 0, 1:
		if !cmn.IsInArray(staking_subcommands, subcommand) {
			for _, sc := range staking_subcommands {
				if input == "" || strings.Contains(sc, subcommand) {
					options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
				}
			}
			return "action", &options, subcommand
		}
	case 2:
		if subcommand == "remove" || subcommand == "edit" || subcommand == "discover" {
			for _, chain := range w.Blockchains {
				if cmn.Contains(chain.Name, bchain) {
					options = append(options, ui.ACOption{
						Name:   chain.Name,
						Result: command + " " + subcommand + " '" + chain.Name + "' "})
				}
			}
			return "blockchain", &options, bchain
		}
		if subcommand == "add" {
			// Show predefined staking providers for matching chains
			for _, s := range cmn.PredefinedStakings {
				b := w.GetBlockchain(s.ChainId)
				if b != nil && cmn.Contains(s.Name, bchain) {
					options = append(options, ui.ACOption{
						Name:   s.Name + " (" + b.Name + ")",
						Result: command + " add '" + s.Name + "' "})
				}
			}
			// Also allow selecting chain for custom staking
			for _, chain := range w.Blockchains {
				if cmn.Contains(chain.Name, bchain) {
					options = append(options, ui.ACOption{
						Name:   "(custom) " + chain.Name,
						Result: command + " add custom '" + chain.Name + "' "})
				}
			}
			return "staking", &options, bchain
		}
	case 3:
		if subcommand == "edit" || subcommand == "remove" || subcommand == "discover" {
			b := w.GetBlockchainByName(bchain)
			if b != nil {
				for _, s := range w.Stakings {
					if s.ChainId == b.ChainId && cmn.Contains(s.Name, name) {
						options = append(options, ui.ACOption{
							Name:   s.Name,
							Result: command + " " + subcommand + " '" + b.Name + "' '" + s.Name + "' "})
					}
				}
			}
			return "provider", &options, name
		}
	case 4:
		if subcommand == "discover" {
			for _, addr := range w.Addresses {
				addrHex := addr.Address.Hex()
				if cmn.Contains(addrHex, addrStr) || cmn.Contains(addr.Name, addrStr) {
					options = append(options, ui.ACOption{
						Name:   addr.Name + " (" + cmn.ShortAddress(addr.Address) + ")",
						Result: command + " " + subcommand + " '" + bchain + "' '" + name + "' " + addrHex})
				}
			}
			return "address", &options, addrStr
		}
	}
	return "", &options, ""
}

func Staking_Process(c *Command, input string) {
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
	_, subcommand, chain, name, addrFilter := p[0], p[1], p[2], p[3], p[4]

	switch subcommand {
	case "list", "":
		listStakingPositions(w)

	case "providers":
		ui.Printf("\nStaking Providers\n\n")

		if len(w.Stakings) == 0 {
			ui.Printf("(no staking providers)\n")
		} else {
			sort.Slice(w.Stakings, func(i, j int) bool {
				if w.Stakings[i].ChainId == w.Stakings[j].ChainId {
					return w.Stakings[i].Name < w.Stakings[j].Name
				}
				return w.Stakings[i].ChainId < w.Stakings[j].ChainId
			})

			for i, s := range w.Stakings {
				b := w.GetBlockchain(s.ChainId)
				if b == nil {
					ui.PrintErrorf("Staking_Process: Blockchain not found: %d", s.ChainId)
					continue
				}

				stakedToken := w.GetTokenByAddress(s.ChainId, s.StakedToken)
				stakedSymbol := "???"
				if stakedToken != nil {
					stakedSymbol = stakedToken.Symbol
				}

				ui.Printf("%d %-12s ", i+1, b.Name)

				// Name as link if URL is set
				if s.URL != "" {
					ui.Terminal.Screen.AddLink(fmt.Sprintf("%-20s", s.Name), "open "+s.URL, s.URL, "")
				} else {
					ui.Printf("%-20s ", s.Name)
				}

				ui.Printf("Staked: %-8s ", stakedSymbol)
				ui.Printf("Rewards: ")

				// Reward 1
				if s.Reward1Token != (common.Address{}) {
					rt := w.GetTokenByAddress(s.ChainId, s.Reward1Token)
					if rt != nil {
						ui.Printf("%s", rt.Symbol)
					} else {
						ui.Printf("???")
					}
				}

				// Reward 2
				if s.Reward2Token != (common.Address{}) {
					ui.Printf(",")
					rt := w.GetTokenByAddress(s.ChainId, s.Reward2Token)
					if rt != nil {
						ui.Printf("%s", rt.Symbol)
					} else {
						ui.Printf("???")
					}
				}

				ui.Printf(" ")
				ui.Terminal.Screen.AddLink(cmn.ICON_EDIT, "command staking edit "+strconv.Itoa(s.ChainId)+" '"+s.Name+"'", "Edit staking", "")
				ui.Terminal.Screen.AddLink(cmn.ICON_DELETE, "command staking remove "+strconv.Itoa(s.ChainId)+" '"+s.Name+"'", "Remove staking", "")
				if s.URL != "" {
					ui.Terminal.Screen.AddLink(cmn.ICON_LINK, "open "+s.URL, s.URL, "")
				}
				cmn.AddAddressShortLink(ui.Terminal.Screen, s.Contract)
				ui.Printf("\n")
			}
		}

		ui.Printf("\n")

	case "add":
		if chain == "" {
			ui.PrintErrorf("Usage: staking add [NAME or custom CHAIN]")
			break
		}

		// Check if it's a predefined staking
		if chain != "custom" {
			for _, s := range cmn.PredefinedStakings {
				if s.Name == chain {
					b := w.GetBlockchain(s.ChainId)
					if b == nil {
						err = fmt.Errorf("blockchain not configured for chain id %d", s.ChainId)
						break
					}

					// Check if already exists
					existing := w.GetStaking(s.ChainId, s.Contract)
					if existing != nil {
						ui.PrintErrorf("Staking %s already added", s.Name)
						break
					}

					stakingCopy := s
					err = w.AddStaking(&stakingCopy)
					if err != nil {
						ui.PrintErrorf("Failed to add staking: %s", err)
						break
					}
					ui.Printf("Staking %s added\n", s.Name)
					break
				}
			}
			break
		}

		// Custom staking - chain is in 'name' parameter
		b := w.GetBlockchainByName(name)
		if b == nil {
			if name == "" {
				ui.PrintErrorf("Usage: staking add custom [CHAIN]")
			} else {
				err = fmt.Errorf("blockchain not found: %s", name)
			}
			break
		}

		bus.Send("ui", "popup", ui.DlgStakingAdd(b))

	case "edit":
		b := w.GetBlockchainByName(chain)
		if b == nil {
			err = fmt.Errorf("blockchain not found: %s", chain)
			break
		}

		s := w.GetStakingByName(b.ChainId, name)
		if s == nil {
			s = w.GetStaking(b.ChainId, common.HexToAddress(name))
		}
		if s == nil {
			err = fmt.Errorf("staking not found: %s", name)
			break
		}

		bus.Send("ui", "popup", ui.DlgStakingEdit(b, s))

	case "remove":
		b := w.GetBlockchainByName(chain)
		if b == nil {
			err = fmt.Errorf("blockchain not found: %s", chain)
			break
		}

		s := w.GetStakingByName(b.ChainId, name)
		if s == nil {
			s = w.GetStaking(b.ChainId, common.HexToAddress(name))
			if s == nil {
				err = fmt.Errorf("staking not found: %s", name)
				break
			}
		}

		bus.Send("ui", "popup", ui.DlgConfirm(
			"Remove staking",
			`
<c>Are you sure you want to remove staking?</c>

       Name:`+s.Name+`
 Blockchain:`+b.Name+`
   Contract:`+s.Contract.String()+`
`,
			func() bool {
				err := w.RemoveStaking(b.ChainId, s.Contract)
				if err != nil {
					ui.PrintErrorf("Error removing staking: %v", err)
					return false
				}
				ui.Notification.Show("Staking removed")
				return true
			}))

	case "discover":
		if len(w.Stakings) == 0 {
			ui.Printf("No staking providers configured\n")
			break
		}

		// Optional filters: chain, provider, and address
		var filterChainId int
		var filterProvider string
		var filterAddress common.Address

		if chain != "" {
			b := w.GetBlockchainByName(chain)
			if b == nil {
				err = fmt.Errorf("blockchain not found: %s", chain)
				break
			}
			filterChainId = b.ChainId

			if name != "" {
				filterProvider = name
			}

			if addrFilter != "" {
				if !common.IsHexAddress(addrFilter) {
					err = fmt.Errorf("invalid address: %s", addrFilter)
					break
				}
				filterAddress = common.HexToAddress(addrFilter)
			}
		}

		added := 0
		removed := 0
		checked := 0

		for _, s := range w.Stakings {
			// Apply chain filter
			if filterChainId != 0 && s.ChainId != filterChainId {
				continue
			}
			// Apply provider filter
			if filterProvider != "" && s.Name != filterProvider {
				continue
			}

			log.Trace().Str("provider", s.Name).Int("chain", s.ChainId).Uint64("validatorId", s.ValidatorId).Msg("Discover: checking provider")

			// Handle validator-based staking (e.g., Monad native staking) specially
			if s.BalanceFunc == "getDelegator" && s.ValidatorId == 0 {
				// Query each address for their delegations
				for _, addr := range w.Addresses {
					// Apply address filter
					if filterAddress != (common.Address{}) && addr.Address != filterAddress {
						continue
					}
					checked++
					log.Trace().Str("provider", s.Name).Str("addr", addr.Address.Hex()).Msg("Discover: querying delegations")

					resp := bus.Fetch("staking", "get-delegations", &bus.B_Staking_GetDelegations{
						ChainId:  s.ChainId,
						Contract: s.Contract,
						Owner:    addr.Address,
					})

					if resp.Error != nil {
						log.Trace().Err(resp.Error).Str("provider", s.Name).Str("addr", addr.Address.Hex()).Msg("Discover: get-delegations error")
						continue
					}

					delegations, ok := resp.Data.(*bus.B_Staking_GetDelegations_Response)
					if !ok || delegations == nil {
						continue
					}

					log.Trace().Str("addr", addr.Address.Hex()).Int("validators", len(delegations.ValidatorIds)).Msg("Discover: found validators")

					for _, valId := range delegations.ValidatorIds {
						// Add position with ValidatorId
						existing := w.GetStakingPositionWithValidator(s.ChainId, s.Contract, addr.Address, valId)
						if existing == nil {
							err := w.AddStakingPosition(&cmn.StakingPosition{
								Owner:       addr.Address,
								ChainId:     s.ChainId,
								Contract:    s.Contract,
								ValidatorId: valId,
							})
							if err == nil {
								added++
								log.Trace().Str("addr", addr.Address.Hex()).Uint64("validatorId", valId).Msg("Discover: added position")
							}
						}
					}
				}
				continue
			}

			for _, addr := range w.Addresses {
				// Apply address filter
				if filterAddress != (common.Address{}) && addr.Address != filterAddress {
					continue
				}
				checked++
				log.Trace().Str("provider", s.Name).Str("addr", addr.Address.Hex()).Msg("Discover: checking address")

				resp := bus.Fetch("staking", "get-balance", &bus.B_Staking_GetBalance{
					ChainId:  s.ChainId,
					Contract: s.Contract,
					Owner:    addr.Address,
				})

				hasBalance := false
				if resp.Error != nil {
					log.Trace().Err(resp.Error).Str("provider", s.Name).Str("addr", addr.Address.Hex()).Msg("Discover: get-balance error")
				} else {
					if balance, ok := resp.Data.(*bus.B_Staking_GetBalance_Response); ok && balance.Balance != nil {
						hasBalance = balance.Balance.Sign() > 0
						log.Trace().Str("provider", s.Name).Str("addr", addr.Address.Hex()).Str("balance", balance.Balance.String()).Bool("hasBalance", hasBalance).Msg("Discover: balance result")
					}
				}

				existing := w.GetStakingPosition(s.ChainId, s.Contract, addr.Address)

				if hasBalance && existing == nil {
					// Add new position
					err := w.AddStakingPosition(&cmn.StakingPosition{
						Owner:    addr.Address,
						ChainId:  s.ChainId,
						Contract: s.Contract,
					})
					if err == nil {
						added++
					}
				} else if !hasBalance && existing != nil {
					// Remove position with 0 balance
					err := w.RemoveStakingPosition(s.ChainId, s.Contract, addr.Address)
					if err == nil {
						removed++
					}
				}
			}
		}
		ui.Printf("Checked %d, positions: %d added, %d removed\n", checked, added, removed)

	case "on":
		w.StakingPaneOn = true
		err = w.Save()
		if err == nil {
			ui.Printf("Staking pane enabled\n")
		}

	case "off":
		w.StakingPaneOn = false
		err = w.Save()
		if err == nil {
			ui.Printf("Staking pane disabled\n")
		}

	default:
		err = fmt.Errorf("unknown command: %s", subcommand)
	}

	if err != nil {
		ui.PrintErrorf(err.Error())
	}
}

func listStakingPositions(w *cmn.Wallet) {
	ui.Printf("\nStaking Positions\n\n")

	if len(w.StakingPositions) == 0 {
		ui.Printf("(no positions)\n")
		return
	}

	const providerWidth = 14
	const tokenWidth = 6
	const valueWidth = 10
	const dollarWidth = 10

	for _, pos := range w.StakingPositions {
		s := w.GetStaking(pos.ChainId, pos.Contract)
		if s == nil {
			continue
		}

		b := w.GetBlockchain(pos.ChainId)
		if b == nil {
			continue
		}

		owner := w.GetAddress(pos.Owner)
		if owner == nil {
			continue
		}

		stakedToken := w.GetTokenByAddress(s.ChainId, s.StakedToken)
		// For native token staking, get the native token
		if stakedToken == nil && s.StakedToken == (common.Address{}) {
			stakedToken, _ = w.GetNativeToken(b)
		}

		// Get staked balance
		balResp := bus.Fetch("staking", "get-balance", &bus.B_Staking_GetBalance{
			ChainId:     s.ChainId,
			Contract:    s.Contract,
			Owner:       pos.Owner,
			ValidatorId: pos.ValidatorId,
		})

		// Provider name as link if URL is set (fixed width)
		displayName := s.Name
		if pos.ValidatorId > 0 {
			displayName = fmt.Sprintf("%s #%d", s.Name, pos.ValidatorId)
		}
		providerName := cmn.FixedWidth(displayName, providerWidth)
		if s.URL != "" {
			ui.Terminal.Screen.AddLink(providerName, "open "+s.URL, s.URL, "")
		} else {
			ui.Printf("%s", providerName)
		}
		ui.Printf(" ")

		// Staked token symbol and balance
		if stakedToken != nil {
			ui.Printf("%s ", cmn.FixedWidth(stakedToken.Symbol, tokenWidth))
			if balResp.Error == nil {
				if balance, ok := balResp.Data.(*bus.B_Staking_GetBalance_Response); ok && balance.Balance != nil {
					cmn.AddFixedValueLink(ui.Terminal.Screen, balance.Balance, stakedToken, valueWidth)
					cmn.AddFixedDollarValueLink(ui.Terminal.Screen, balance.Balance, stakedToken, dollarWidth)
				}
			}
		} else {
			ui.Terminal.Screen.AddLink(cmn.FixedWidth("???", tokenWidth), "command token add "+b.Name+" "+s.StakedToken.String(), "Add token", "")
			ui.Printf(" ")
			if balResp.Error == nil {
				if balance, ok := balResp.Data.(*bus.B_Staking_GetBalance_Response); ok && balance.Balance != nil {
					xf := cmn.NewXF(balance.Balance, 18)
					ui.Terminal.Screen.AddLink(fmt.Sprintf("%*s", valueWidth, cmn.FmtAmount(balance.Balance, 18, true)), "copy "+xf.String(), xf.String(), "")
					ui.Printf("%*s", dollarWidth, cmn.FmtFloat64D(0, true))
				}
			}
		}

		// Reward 1
		if s.Reward1Func != "" {
			rewardToken := w.GetTokenByAddress(s.ChainId, s.Reward1Token)
			// For native token rewards, get the native token
			if rewardToken == nil && s.Reward1Token == (common.Address{}) {
				rewardToken, _ = w.GetNativeToken(b)
			}

			pendingResp := bus.Fetch("staking", "get-pending", &bus.B_Staking_GetPending{
				ChainId:     s.ChainId,
				Contract:    s.Contract,
				Owner:       pos.Owner,
				RewardToken: s.Reward1Token,
				ValidatorId: pos.ValidatorId,
			})

			if pendingResp.Error == nil {
				if pending, ok := pendingResp.Data.(*bus.B_Staking_GetPending_Response); ok && pending.Pending != nil {
					if rewardToken != nil {
						ui.Printf(" %s:", cmn.FixedWidth(rewardToken.Symbol, tokenWidth))
						cmn.AddFixedValueLink(ui.Terminal.Screen, pending.Pending, rewardToken, valueWidth)
						cmn.AddFixedDollarValueLink(ui.Terminal.Screen, pending.Pending, rewardToken, dollarWidth)
					} else {
						ui.Printf(" ")
						ui.Terminal.Screen.AddLink(cmn.FixedWidth("???", tokenWidth)+":", "command token add "+b.Name+" "+s.Reward1Token.String(), "Add token", "")
						xf := cmn.NewXF(pending.Pending, 18)
						ui.Terminal.Screen.AddLink(fmt.Sprintf("%*s", valueWidth, cmn.FmtAmount(pending.Pending, 18, true)), "copy "+xf.String(), xf.String(), "")
						ui.Printf("%*s", dollarWidth, cmn.FmtFloat64D(0, true))
					}
				}
			}
		}

		// Reward 2
		if s.Reward2Func != "" {
			rewardToken := w.GetTokenByAddress(s.ChainId, s.Reward2Token)
			// For native token rewards, get the native token
			if rewardToken == nil && s.Reward2Token == (common.Address{}) {
				rewardToken, _ = w.GetNativeToken(b)
			}

			pendingResp := bus.Fetch("staking", "get-pending", &bus.B_Staking_GetPending{
				ChainId:     s.ChainId,
				Contract:    s.Contract,
				Owner:       pos.Owner,
				RewardToken: s.Reward2Token,
				ValidatorId: pos.ValidatorId,
			})

			if pendingResp.Error == nil {
				if pending, ok := pendingResp.Data.(*bus.B_Staking_GetPending_Response); ok && pending.Pending != nil {
					if rewardToken != nil {
						ui.Printf(" %s:", cmn.FixedWidth(rewardToken.Symbol, tokenWidth))
						cmn.AddFixedValueLink(ui.Terminal.Screen, pending.Pending, rewardToken, valueWidth)
						cmn.AddFixedDollarValueLink(ui.Terminal.Screen, pending.Pending, rewardToken, dollarWidth)
					} else {
						ui.Printf(" ")
						ui.Terminal.Screen.AddLink(cmn.FixedWidth("???", tokenWidth)+":", "command token add "+b.Name+" "+s.Reward2Token.String(), "Add token", "")
						xf := cmn.NewXF(pending.Pending, 18)
						ui.Terminal.Screen.AddLink(fmt.Sprintf("%*s", valueWidth, cmn.FmtAmount(pending.Pending, 18, true)), "copy "+xf.String(), xf.String(), "")
						ui.Printf("%*s", dollarWidth, cmn.FmtFloat64D(0, true))
					}
				}
			}
		}

		ui.Printf(" %s\n", owner.Name)
		ui.Flush()
	}

	ui.Printf("\n")
}
