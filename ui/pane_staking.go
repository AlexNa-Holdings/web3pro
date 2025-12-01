package ui

import (
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

type StakingPane struct {
	PaneDescriptor
	On bool
}

type stakingPositionInfo struct {
	Position     *cmn.StakingPosition
	Staking      *cmn.Staking
	Owner        *cmn.Address
	StakedAmount *big.Int
	StakedUSD    float64
	Reward1      *big.Int
	Reward1USD   float64
	Reward2      *big.Int
	Reward2USD   float64
	TotalUSD     float64
}

var staking_info_list []*stakingPositionInfo = make([]*stakingPositionInfo, 0)

var Staking StakingPane = StakingPane{
	PaneDescriptor: PaneDescriptor{
		MinWidth:               80,
		MinHeight:              2,
		MaxHeight:              20,
		SupportCachedHightCalc: true,
	},
}

func (p *StakingPane) IsOn() bool {
	return p.On
}

func (p *StakingPane) SetOn(on bool) {
	wasOff := !p.On
	p.On = on
	if on && wasOff {
		go p.updateList()
	}
}

func (p *StakingPane) GetDesc() *PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *StakingPane) EstimateLines(w int) int {
	return gocui.EstimateTemplateLines(p.GetTemplate(), w)
}

func (p *StakingPane) SetView(x0, y0, x1, y1 int, overlap byte) {
	v, err := Gui.SetView("staking", x0, y0, x1, y1, overlap)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}

		p.PaneDescriptor.View = v
		v.JoinedFrame = true
		v.Title = "Staking"
		v.ScrollBar = true
		v.OnResize = func(v *gocui.View) {
			v.RenderTemplate(p.GetTemplate())
			v.ScrollTop()
		}
		v.OnOverHotspot = ProcessOnOverHotspot
		v.OnClickHotspot = ProcessOnClickHotspot

		p.SetTemplate(p.rebuildTemplate())
		v.RenderTemplate(p.GetTemplate())
	}
}

func StakingLoop() {
	ch := bus.Subscribe("wallet", "price", "staking")
	defer bus.Unsubscribe(ch)

	for msg := range ch {
		go Staking.processStaking(msg)
	}
}

func (p *StakingPane) processStaking(msg *bus.Message) {
	switch msg.Topic {
	case "wallet":
		switch msg.Type {
		case "open", "saved":
			p.updateList()
		}
	case "price":
		switch msg.Type {
		case "updated":
			p.updateList()
		}
	case "staking":
		switch msg.Type {
		case "updated":
			p.updateList()
		}
	}
}

func (p *StakingPane) updateList() {
	log.Trace().Msg("Staking: updateList called")

	if !p.On {
		log.Trace().Msg("Staking: pane is off, skipping")
		return
	}

	w := cmn.CurrentWallet
	if w == nil {
		log.Trace().Msg("Staking: no wallet, skipping")
		return
	}

	log.Trace().Int("positions", len(w.StakingPositions)).Msg("Staking: checking positions")

	if len(w.StakingPositions) == 0 {
		staking_info_list = make([]*stakingPositionInfo, 0)
		if Staking.View != nil {
			Gui.Update(func(g *gocui.Gui) error {
				if Staking.View != nil {
					p.SetTemplate(Staking.rebuildTemplate())
					Staking.View.RenderTemplate(p.GetTemplate())
				}
				return nil
			})
		}
		return
	}

	list := make([]*stakingPositionInfo, 0)
	totalStakedUSD := 0.0
	totalRewardsUSD := 0.0

	for _, pos := range w.StakingPositions {
		s := w.GetStaking(pos.ChainId, pos.Contract)
		if s == nil {
			log.Trace().Str("contract", pos.Contract.Hex()).Msg("Staking: staking not found for position")
			continue
		}

		owner := w.GetAddress(pos.Owner)
		if owner == nil {
			log.Trace().Str("owner", pos.Owner.Hex()).Msg("Staking: owner not found for position")
			continue
		}

		info := &stakingPositionInfo{
			Position: pos,
			Staking:  s,
			Owner:    owner,
		}

		// Get staked balance
		stakedToken := w.GetTokenByAddress(s.ChainId, s.StakedToken)
		log.Trace().Str("contract", s.Contract.Hex()).Str("owner", pos.Owner.Hex()).Uint64("validatorId", pos.ValidatorId).Msg("Staking: fetching balance")
		balResp := bus.Fetch("staking", "get-balance", &bus.B_Staking_GetBalance{
			ChainId:     s.ChainId,
			Contract:    s.Contract,
			Owner:       pos.Owner,
			ValidatorId: pos.ValidatorId,
		})

		if balResp.Error != nil {
			log.Trace().Err(balResp.Error).Msg("Staking: get-balance failed")
		} else {
			if balance, ok := balResp.Data.(*bus.B_Staking_GetBalance_Response); ok && balance.Balance != nil {
				info.StakedAmount = balance.Balance
				if stakedToken != nil {
					info.StakedUSD = stakedToken.Price * stakedToken.Float64(balance.Balance)
				}
			}
		}

		// Get reward 1
		// Note: For native token rewards (e.g., Monad), Reward1Token may be zero address
		log.Trace().Str("provider", s.Name).Str("Reward1Func", s.Reward1Func).Str("Reward1Token", s.Reward1Token.Hex()).Msg("Staking: checking rewards")
		if s.Reward1Func != "" {
			rewardToken := w.GetTokenByAddress(s.ChainId, s.Reward1Token)
			// For native token rewards, get the native token
			if rewardToken == nil && s.Reward1Token == ([20]byte{}) {
				b := w.GetBlockchain(s.ChainId)
				if b != nil {
					rewardToken, _ = w.GetNativeToken(b)
				}
			}
			log.Trace().Str("provider", s.Name).Uint64("validatorId", pos.ValidatorId).Msg("Staking: fetching pending rewards")
			pendingResp := bus.Fetch("staking", "get-pending", &bus.B_Staking_GetPending{
				ChainId:     s.ChainId,
				Contract:    s.Contract,
				Owner:       pos.Owner,
				RewardToken: s.Reward1Token,
				ValidatorId: pos.ValidatorId,
			})

			if pendingResp.Error != nil {
				log.Trace().Err(pendingResp.Error).Str("provider", s.Name).Msg("Staking: get-pending error")
			} else {
				if pending, ok := pendingResp.Data.(*bus.B_Staking_GetPending_Response); ok && pending.Pending != nil {
					info.Reward1 = pending.Pending
					log.Trace().Str("provider", s.Name).Str("pending", pending.Pending.String()).Msg("Staking: got pending rewards")
					if rewardToken != nil {
						info.Reward1USD = rewardToken.Price * rewardToken.Float64(pending.Pending)
					}
				}
			}
		}

		// Get reward 2
		if s.Reward2Token != ([20]byte{}) && s.Reward2Func != "" {
			rewardToken := w.GetTokenByAddress(s.ChainId, s.Reward2Token)
			pendingResp := bus.Fetch("staking", "get-pending", &bus.B_Staking_GetPending{
				ChainId:     s.ChainId,
				Contract:    s.Contract,
				Owner:       pos.Owner,
				RewardToken: s.Reward2Token,
				ValidatorId: pos.ValidatorId,
			})

			if pendingResp.Error == nil {
				if pending, ok := pendingResp.Data.(*bus.B_Staking_GetPending_Response); ok && pending.Pending != nil {
					info.Reward2 = pending.Pending
					if rewardToken != nil {
						info.Reward2USD = rewardToken.Price * rewardToken.Float64(pending.Pending)
					}
				}
			}
		}

		info.TotalUSD = info.StakedUSD + info.Reward1USD + info.Reward2USD
		totalStakedUSD += info.StakedUSD
		totalRewardsUSD += info.Reward1USD + info.Reward2USD

		list = append(list, info)
	}

	// Sort by total USD value descending
	sort.Slice(list, func(i, j int) bool {
		return list[i].TotalUSD > list[j].TotalUSD
	})

	log.Trace().Int("list_size", len(list)).Msg("Staking: updateList completed")

	staking_info_list = list

	if Staking.View != nil {
		Gui.Update(func(g *gocui.Gui) error {
			if Staking.View != nil {
				p.SetTemplate(Staking.rebuildTemplate())
				Staking.View.RenderTemplate(p.GetTemplate())
				Staking.View.ScrollTop()
			}
			return nil
		})
	}

	if Staking.View != nil {
		Staking.View.Subtitle = fmt.Sprintf("N:%d %s \uf0d8%s", len(list),
			cmn.FmtFloat64D(totalStakedUSD, true),
			cmn.FmtFloat64D(totalRewardsUSD, true))
	}
}

func (p *StakingPane) rebuildTemplate() string {
	w := cmn.CurrentWallet
	if w == nil {
		return "no open wallet"
	}

	if len(w.StakingPositions) == 0 {
		return "(no positions)"
	}

	if len(staking_info_list) == 0 {
		return "loading..."
	}

	const providerWidth = 18
	const tokenWidth = 6
	const valueWidth = 10
	const dollarWidth = 10

	temp := ""

	for i, info := range staking_info_list {
		s := info.Staking
		pos := info.Position
		stakedToken := w.GetTokenByAddress(s.ChainId, s.StakedToken)

		// Provider name as link (fixed width)
		providerName := s.Name
		if pos.ValidatorId > 0 {
			providerName = fmt.Sprintf("%s #%d", s.Name, pos.ValidatorId)
		}
		providerName = cmn.FixedWidth(providerName, providerWidth)
		if s.URL != "" {
			temp += fmt.Sprintf("<l text:'%s' action:'open %s' tip:'%s'>", providerName, s.URL, s.URL)
		} else {
			temp += providerName
		}
		temp += " "

		// Staked token symbol and amount
		b := w.GetBlockchain(s.ChainId)
		// For native token staking, get the native token
		if stakedToken == nil && s.StakedToken == ([20]byte{}) && b != nil {
			stakedToken, _ = w.GetNativeToken(b)
		}
		if stakedToken != nil {
			temp += cmn.FixedWidth(stakedToken.Symbol, tokenWidth) + " "
			if info.StakedAmount != nil {
				temp += cmn.TagFixedValueLink(info.StakedAmount, stakedToken, valueWidth)
				temp += cmn.TagFixedDollarLink(info.StakedUSD, dollarWidth)
			}
		} else if b != nil {
			temp += cmn.TagLink(cmn.FixedWidth("???", tokenWidth), "command token add "+b.Name+" "+s.StakedToken.String(), "Add token") + " "
			if info.StakedAmount != nil {
				xf := cmn.NewXF(info.StakedAmount, 18)
				temp += fmt.Sprintf("<l text:'%*s' action:'copy %s' tip:'%s'>", valueWidth, cmn.FmtAmount(info.StakedAmount, 18, true), xf.String(), xf.String())
				temp += fmt.Sprintf("%*s", dollarWidth, cmn.FmtFloat64D(0, true))
			}
		}

		// Reward 1
		if s.Reward1Func != "" {
			rewardToken := w.GetTokenByAddress(s.ChainId, s.Reward1Token)
			// For native token rewards, get the native token
			if rewardToken == nil && s.Reward1Token == ([20]byte{}) && b != nil {
				rewardToken, _ = w.GetNativeToken(b)
			}
			if rewardToken != nil {
				temp += " " + cmn.FixedWidth(rewardToken.Symbol, tokenWidth) + ":"
				if info.Reward1 != nil {
					temp += cmn.TagFixedValueLink(info.Reward1, rewardToken, valueWidth)
				} else {
					temp += fmt.Sprintf("%*s", valueWidth, "0")
				}
				temp += cmn.TagFixedDollarLink(info.Reward1USD, dollarWidth)
			} else if b != nil {
				temp += " " + cmn.TagLink(cmn.FixedWidth("???", tokenWidth)+":", "command token add "+b.Name+" "+s.Reward1Token.String(), "Add token")
				if info.Reward1 != nil {
					xf := cmn.NewXF(info.Reward1, 18)
					temp += fmt.Sprintf("<l text:'%*s' action:'copy %s' tip:'%s'>", valueWidth, cmn.FmtAmount(info.Reward1, 18, true), xf.String(), xf.String())
				} else {
					temp += fmt.Sprintf("%*s", valueWidth, "0")
				}
				temp += fmt.Sprintf("%*s", dollarWidth, cmn.FmtFloat64D(0, true))
			}
		}

		// Reward 2
		if s.Reward2Func != "" {
			rewardToken := w.GetTokenByAddress(s.ChainId, s.Reward2Token)
			// For native token rewards, get the native token
			if rewardToken == nil && s.Reward2Token == ([20]byte{}) && b != nil {
				rewardToken, _ = w.GetNativeToken(b)
			}
			if rewardToken != nil {
				temp += " " + cmn.FixedWidth(rewardToken.Symbol, tokenWidth) + ":"
				if info.Reward2 != nil {
					temp += cmn.TagFixedValueLink(info.Reward2, rewardToken, valueWidth)
				} else {
					temp += fmt.Sprintf("%*s", valueWidth, "0")
				}
				temp += cmn.TagFixedDollarLink(info.Reward2USD, dollarWidth)
			} else if b != nil {
				temp += " " + cmn.TagLink(cmn.FixedWidth("???", tokenWidth)+":", "command token add "+b.Name+" "+s.Reward2Token.String(), "Add token")
				if info.Reward2 != nil {
					xf := cmn.NewXF(info.Reward2, 18)
					temp += fmt.Sprintf("<l text:'%*s' action:'copy %s' tip:'%s'>", valueWidth, cmn.FmtAmount(info.Reward2, 18, true), xf.String(), xf.String())
				} else {
					temp += fmt.Sprintf("%*s", valueWidth, "0")
				}
				temp += fmt.Sprintf("%*s", dollarWidth, cmn.FmtFloat64D(0, true))
			}
		}

		temp += " " + info.Owner.Name

		if i < len(staking_info_list)-1 {
			temp += "\n"
		}
	}

	return temp
}
