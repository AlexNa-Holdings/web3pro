package ui

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

type LP_V4Pane struct {
	PaneDescriptor
	On bool
}

var lp_v4_info_list []*bus.B_LP_V4_GetPositionStatus_Response = make([]*bus.B_LP_V4_GetPositionStatus_Response, 0)

var lpV4UpdateMu sync.Mutex
var lpV4UpdatePending bool
var lpV4LastUpdate time.Time

var LP_V4 LP_V4Pane = LP_V4Pane{
	PaneDescriptor: PaneDescriptor{
		MinWidth:               90,
		MinHeight:              2,
		MaxHeight:              30,
		SupportCachedHightCalc: true,
	},
}

func (p *LP_V4Pane) IsOn() bool {
	return p.On
}

func (p *LP_V4Pane) SetOn(on bool) {
	p.On = on
}

func (p *LP_V4Pane) GetDesc() *PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *LP_V4Pane) EstimateLines(w int) int {
	return gocui.EstimateTemplateLines(p.GetTemplate(), w)
}

func (p *LP_V4Pane) SetView(x0, y0, x1, y1 int, overlap byte) {
	v, err := Gui.SetView("v4", x0, y0, x1, y1, overlap)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}

		p.PaneDescriptor.View = v
		v.JoinedFrame = true
		v.Title = "LP v4"
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

func LP_V4Loop() {
	ch := bus.Subscribe("wallet", "price", "eth")
	defer bus.Unsubscribe(ch)

	for msg := range ch {
		go LP_V4.processV4(msg)
	}
}

func (p *LP_V4Pane) processV4(msg *bus.Message) {
	switch msg.Topic {
	case "wallet":
		switch msg.Type {
		case "saved":
			p.scheduleUpdate()
		}
	case "price":
		switch msg.Type {
		case "updated":
			p.scheduleUpdate()
		}
	case "eth":
		switch msg.Type {
		case "connected":
			p.scheduleUpdate()
		}
	}
}

// scheduleUpdate debounces updateList calls to avoid flooding RPC on startup
func (p *LP_V4Pane) scheduleUpdate() {
	lpV4UpdateMu.Lock()
	defer lpV4UpdateMu.Unlock()

	// If update is already pending, skip
	if lpV4UpdatePending {
		return
	}

	// Debounce: wait at least 2 seconds between updates
	timeSinceLastUpdate := time.Since(lpV4LastUpdate)
	if timeSinceLastUpdate < 2*time.Second {
		lpV4UpdatePending = true
		go func() {
			time.Sleep(2*time.Second - timeSinceLastUpdate)
			lpV4UpdateMu.Lock()
			lpV4UpdatePending = false
			lpV4LastUpdate = time.Now()
			lpV4UpdateMu.Unlock()
			p.updateList()
		}()
		return
	}

	lpV4LastUpdate = time.Now()
	go p.updateList()
}

func (p *LP_V4Pane) updateList() {
	if !p.On {
		return
	}

	w := cmn.CurrentWallet
	if w == nil {
		return
	}

	if len(w.LP_V4_Positions) == 0 {
		return
	}

	total_gain := 0.0
	total_liq := 0.0

	list := make([]*bus.B_LP_V4_GetPositionStatus_Response, 0)
	to_delete := make([]*cmn.LP_V4_Position, 0)

	for _, pos := range w.LP_V4_Positions {
		sr := bus.Fetch("lp_v4", "get-position-status", &bus.B_LP_V4_GetPositionStatus{
			ChainId:   pos.ChainId,
			Provider:  pos.Provider,
			NFT_Token: pos.NFT_Token,
		})

		if sr.Error != nil {
			log.Error().Err(sr.Error).Msg("get_position_status")
			continue
		}

		resp, ok := sr.Data.(*bus.B_LP_V4_GetPositionStatus_Response)
		if !ok {
			log.Error().Msg("get_position_status")
			continue
		}

		big0 := big.NewInt(0)
		// Remove positions with 0 liquidity and 0 gain (fully closed positions)
		if resp.Liquidity0.Cmp(big0) == 0 && resp.Liquidity1.Cmp(big0) == 0 &&
			resp.Gain0.Cmp(big0) == 0 && resp.Gain1.Cmp(big0) == 0 {
			to_delete = append(to_delete, pos)
			continue
		}

		// Skip positions with ignored tokens
		t0 := w.GetTokenByAddress(resp.ChainId, resp.Currency0)
		t1 := w.GetTokenByAddress(resp.ChainId, resp.Currency1)
		if (t0 != nil && t0.Ignored) || (t1 != nil && t1.Ignored) {
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
			}
			return list[i].ProviderName < list[j].ProviderName
		}
		return list[i].Gain0Dollars+list[i].Gain1Dollars > list[j].Gain0Dollars+list[j].Gain1Dollars
	})

	lp_v4_info_list = list

	if LP_V4.View != nil {
		Gui.Update(func(g *gocui.Gui) error {
			if LP_V4.View != nil {
				p.SetTemplate(LP_V4.rebuidTemplate())
				LP_V4.View.RenderTemplate(p.GetTemplate())
				LP_V4.View.ScrollTop()
			}
			return nil
		})
	}

	if LP_V4.View != nil {
		LP_V4.View.Subtitle = fmt.Sprintf("NPos:%d %s \uf0d8%s", len(list),
			cmn.FmtFloat64D(total_liq, true),
			cmn.FmtFloat64D(total_gain, true))
	}

	// Remove closed positions from wallet
	for _, pos := range to_delete {
		w.RemoveLP_V4Position(pos.Owner, pos.ChainId, pos.Provider, pos.NFT_Token)
	}
}

func (p *LP_V4Pane) rebuidTemplate() string {
	w := cmn.CurrentWallet
	if w == nil {
		return "no open wallet"
	}

	if len(w.LP_V4_Positions) == 0 {
		return "(no positions)"
	}

	if len(lp_v4_info_list) == 0 {
		return "loading..."
	}

	temp := "Xch@Chain        Pair        ID On Liq0     Liq1     Gain0    Gain1     Gain$  Address\n"

	for i, p := range lp_v4_info_list {

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

		temp += cmn.TagLink(
			fmt.Sprintf("%-12s", p.ProviderName),
			"open "+provider.URL,
			"open "+provider.URL)

		t0 := w.GetTokenByAddress(p.ChainId, p.Currency0)
		t1 := w.GetTokenByAddress(p.ChainId, p.Currency1)

		if t0 != nil && t1 != nil {
			temp += fmt.Sprintf("%11s", t0.Symbol+"/"+t1.Symbol)
		} else {
			if t0 != nil {
				temp += fmt.Sprintf("%-5s", t0.Symbol)
			} else {
				temp += cmn.TagLink("???", "command token add "+b.Name+" "+p.Currency0.String(), "Add token")
			}

			temp += "/"

			if t1 != nil {
				temp += fmt.Sprintf("%-5s", t1.Symbol)
			} else {
				temp += cmn.TagLink("???", "command token add "+b.Name+" "+p.Currency1.String(), "Add token")
			}
		}

		// NFT position ID
		if p.NFT_Token != nil {
			idStr := p.NFT_Token.String()
			temp += fmt.Sprintf(" <l text:'%6s' action:'copy %s' tip:'%s'>", idStr, idStr, idStr)
		} else {
			temp += "       "
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

		temp += fmt.Sprintf(" %s", owner.Name)

		if i < len(lp_v4_info_list)-1 {
			temp += "\n"
		}

	}
	return temp
}
