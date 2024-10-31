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

type LP_V3Pane struct {
	PaneDescriptor
	On bool
}

var lp_info_list []*bus.B_LP_V3_GetPositionStatus_Response = make([]*bus.B_LP_V3_GetPositionStatus_Response, 0)

var LP_V3 LP_V3Pane = LP_V3Pane{
	PaneDescriptor: PaneDescriptor{
		MinWidth:               90,
		MinHeight:              2,
		MaxHeight:              30,
		SupportCachedHightCalc: true,
	},
}

func (p *LP_V3Pane) IsOn() bool {
	return p.On
}

func (p *LP_V3Pane) SetOn(on bool) {
	p.On = on
}

func (p *LP_V3Pane) GetDesc() *PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *LP_V3Pane) EstimateLines(w int) int {
	return gocui.EstimateTemplateLines(p.GetTemplate(), w)
}

func (p *LP_V3Pane) SetView(x0, y0, x1, y1 int, overlap byte) {
	v, err := Gui.SetView("v3", x0, y0, x1, y1, overlap)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}

		p.PaneDescriptor.View = v
		v.JoinedFrame = true
		v.Title = "LP v3"
		v.ScrollBar = true
		v.OnResize = func(v *gocui.View) {
			v.RenderTemplate(p.GetTemplate())
			v.ScrollTop()
		}
		v.OnOverHotspot = ProcessOnOverHotspot
		v.OnClickHotspot = ProcessOnClickHotspot

		p.SetTemplate(p.rebuidTemplate())
		v.RenderTemplate(p.GetTemplate())
	}
}

func LP_V3Loop() {
	ch := bus.Subscribe("wallet", "price")
	defer bus.Unsubscribe(ch)

	for msg := range ch {
		go LP_V3.processV3(msg)
	}
}

func (p *LP_V3Pane) processV3(msg *bus.Message) {
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
	}
}

func (p *LP_V3Pane) updateList() {
	if !p.On {
		return
	}

	w := cmn.CurrentWallet
	if w == nil {
		return
	}

	if len(w.LP_V3_Positions) == 0 {
		return
	}

	total_gain := 0.0
	total_liq := 0.0

	list := make([]*bus.B_LP_V3_GetPositionStatus_Response, 0)
	to_delete := make([]*cmn.LP_V3_Position, 0)

	for _, pos := range w.LP_V3_Positions {
		sr := bus.Fetch("lp_v3", "get-position-status", &bus.B_LP_V3_GetPositionStatus{
			ChainId:   pos.ChainId,
			Provider:  pos.Provider,
			NFT_Token: pos.NFT_Token,
		})

		if sr.Error != nil {
			log.Error().Err(sr.Error).Msg("get_position_status")
			continue
		}

		resp, ok := sr.Data.(*bus.B_LP_V3_GetPositionStatus_Response)
		if !ok {
			log.Error().Msg("get_position_status")
			continue
		}

		big0 := big.NewInt(0)
		if resp.Liquidity0.Cmp(big0) == 0 && resp.Liquidity1.Cmp(big0) == 0 {
			to_delete = append(to_delete, pos)
			continue
		}

		total_gain += resp.Gain0Dollars + resp.Gain1Dollars
		total_liq += resp.Liquidity0Dollars + resp.Liquidity1Dollars
		list = append(list, resp)
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].Gain0Dollars+list[i].Gain1Dollars == list[j].Gain0Dollars+list[j].Gain1Dollars {
			if list[i].ProviderName == list[j].ProviderName {
				return list[i].Owner.Hex() < list[j].Owner.Hex()
			} else {
				return list[i].ProviderName < list[j].ProviderName
			}
		} else {
			if list[i].Gain0Dollars+list[i].Gain1Dollars > list[j].Gain0Dollars+list[j].Gain1Dollars {
				return true
			} else {
				return false
			}
		}
	})

	lp_info_list = list

	if LP_V3.View != nil {
		Gui.Update(func(g *gocui.Gui) error {
			if LP_V3.View != nil {
				p.SetTemplate(LP_V3.rebuidTemplate())
				LP_V3.View.RenderTemplate(p.GetTemplate())
				LP_V3.View.ScrollTop()
			}
			return nil
		})
	}

	if LP_V3.View != nil {
		LP_V3.View.Subtitle = fmt.Sprintf("NPos:%d $%s \uf0d8$%s", len(list),
			cmn.FormatFloatWithCommas(total_liq),
			cmn.FormatFloatWithCommas(total_gain))
	}

	for _, pos := range to_delete {
		w.RemoveLP_V3Position(pos.Owner, pos.ChainId, pos.Provider, pos.NFT_Token)
	}

}

func (p *LP_V3Pane) rebuidTemplate() string {
	w := cmn.CurrentWallet
	if w == nil {
		return "no open wallet"
	}

	if len(w.LP_V3_Positions) == 0 {
		return "(no positions)"
	}

	if len(lp_info_list) == 0 {
		return "loading..."
	}

	temp := "Xch@Chain      Pair   On Liq0     Liq1     Gain0    Gain1     Gain$    Fee%%   Address\n"

	for i, p := range lp_info_list {

		provider := w.GetLP_V3(p.ChainId, p.Provider)
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

		temp += cmn.TagLink(
			fmt.Sprintf("%-12s", p.ProviderName),
			"open "+provider.URL,
			"open "+provider.URL)

		t0 := w.GetTokenByAddress(p.ChainId, p.Token0)
		t1 := w.GetTokenByAddress(p.ChainId, p.Token1)

		if t0 != nil && t1 != nil {
			temp += fmt.Sprintf("%9s", t0.Symbol+"/"+t1.Symbol)
		} else {
			if t0 != nil {
				temp += fmt.Sprintf("%-5s", t0.Symbol)
			} else {
				temp += cmn.TagLink("???", "command token add "+b.Name+" "+p.Token0.String(), "Add token")
			}

			temp += "/"

			if t1 != nil {
				temp += fmt.Sprintf("%-5s", t1.Symbol)
			} else {
				temp += cmn.TagLink("???", "command token add "+b.Name+" "+p.Token1.String(), "Add token")
			}
		}

		if p.On {
			temp += "<color fg:green>" + cmn.ICON_LIGHT + "</color>"
		} else {
			temp += "<color fg:red>" + cmn.ICON_LIGHT + "</color>"
		}

		if t0 != nil {
			temp += cmn.TagValueLink(p.Liquidity0, t0)
		} else {
			temp += "         "
		}

		if t1 != nil {
			temp += cmn.TagValueLink(p.Liquidity1, t1)
		} else {
			temp += "         "
		}

		if t0 != nil {
			temp += cmn.TagValueLink(p.Gain0, t0)
		} else {
			temp += "         "
		}

		if t1 != nil {
			temp += cmn.TagValueLink(p.Gain1, t1)
		} else {
			temp += "         "
		}

		temp += cmn.TagDollarLink(p.Gain0Dollars + p.Gain1Dollars)

		temp += fmt.Sprintf("%2.1f/%2.1f ", p.FeeProtocol0, p.FeeProtocol1)
		temp += fmt.Sprintf(" %s", owner.Name)

		if i < len(lp_info_list)-1 {
			temp += "\n"
		}

	}
	return temp
}
