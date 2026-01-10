package ui

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

type LP_V2Pane struct {
	PaneDescriptor
	On bool
}

var lp_v2_info_list []*bus.B_LP_V2_GetPositionStatus_Response = make([]*bus.B_LP_V2_GetPositionStatus_Response, 0)

var lpV2UpdateMu sync.Mutex
var lpV2UpdatePending bool
var lpV2LastUpdate time.Time

var LP_V2 LP_V2Pane = LP_V2Pane{
	PaneDescriptor: PaneDescriptor{
		MinWidth:               90,
		MinHeight:              2,
		MaxHeight:              30,
		SupportCachedHightCalc: true,
	},
}

func (p *LP_V2Pane) IsOn() bool {
	return p.On
}

func (p *LP_V2Pane) SetOn(on bool) {
	p.On = on
}

func (p *LP_V2Pane) GetDesc() *PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *LP_V2Pane) EstimateLines(w int) int {
	return gocui.EstimateTemplateLines(p.GetTemplate(), w)
}

func (p *LP_V2Pane) SetView(x0, y0, x1, y1 int, overlap byte) {
	v, err := Gui.SetView("v2", x0, y0, x1, y1, overlap)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}

		p.PaneDescriptor.View = v
		v.JoinedFrame = true
		v.Title = "LP v2"
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

func LP_V2Loop() {
	ch := bus.Subscribe("wallet", "price", "eth")
	defer bus.Unsubscribe(ch)

	for msg := range ch {
		go LP_V2.processV2(msg)
	}
}

func (p *LP_V2Pane) processV2(msg *bus.Message) {
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
func (p *LP_V2Pane) scheduleUpdate() {
	lpV2UpdateMu.Lock()
	defer lpV2UpdateMu.Unlock()

	// If update is already pending, skip
	if lpV2UpdatePending {
		return
	}

	// Debounce: wait at least 2 seconds between updates
	timeSinceLastUpdate := time.Since(lpV2LastUpdate)
	if timeSinceLastUpdate < 2*time.Second {
		lpV2UpdatePending = true
		go func() {
			time.Sleep(2*time.Second - timeSinceLastUpdate)
			lpV2UpdateMu.Lock()
			lpV2UpdatePending = false
			lpV2LastUpdate = time.Now()
			lpV2UpdateMu.Unlock()
			p.updateList()
		}()
		return
	}

	lpV2LastUpdate = time.Now()
	go p.updateList()
}

func (p *LP_V2Pane) updateList() {
	if !p.On {
		return
	}

	w := cmn.CurrentWallet
	if w == nil {
		return
	}

	if len(w.LP_V2_Positions) == 0 {
		return
	}

	total_liq := 0.0

	list := make([]*bus.B_LP_V2_GetPositionStatus_Response, 0)

	for _, pos := range w.LP_V2_Positions {
		sr := bus.Fetch("lp_v2", "get-position-status", &bus.B_LP_V2_GetPositionStatus{
			ChainId: pos.ChainId,
			Factory: pos.Factory,
			Pair:    pos.Pair,
		})

		if sr.Error != nil {
			log.Error().Err(sr.Error).Msg("get_position_status")
			continue
		}

		resp, ok := sr.Data.(*bus.B_LP_V2_GetPositionStatus_Response)
		if !ok {
			log.Error().Msg("get_position_status")
			continue
		}

		// Delete positions with 0 liquidity
		if resp.LPBalance == nil || resp.LPBalance.Cmp(big.NewInt(0)) == 0 {
			w.RemoveLP_V2Position(pos.Owner, pos.ChainId, pos.Factory, pos.Pair)
			continue
		}

		// Skip positions with ignored tokens
		t0 := w.GetTokenByAddress(resp.ChainId, resp.Token0)
		t1 := w.GetTokenByAddress(resp.ChainId, resp.Token1)
		if (t0 != nil && t0.Ignored) || (t1 != nil && t1.Ignored) {
			continue
		}

		total_liq += resp.Liquidity0Dollars + resp.Liquidity1Dollars
		list = append(list, resp)
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].Liquidity0Dollars+list[i].Liquidity1Dollars == list[j].Liquidity0Dollars+list[j].Liquidity1Dollars {
			if list[i].ProviderName == list[j].ProviderName {
				return list[i].Owner.Hex() < list[j].Owner.Hex()
			}
			return list[i].ProviderName < list[j].ProviderName
		}
		return list[i].Liquidity0Dollars+list[i].Liquidity1Dollars > list[j].Liquidity0Dollars+list[j].Liquidity1Dollars
	})

	lp_v2_info_list = list

	if LP_V2.View != nil {
		Gui.Update(func(g *gocui.Gui) error {
			if LP_V2.View != nil {
				p.SetTemplate(LP_V2.rebuidTemplate())
				LP_V2.View.RenderTemplate(p.GetTemplate())
				LP_V2.View.ScrollTop()
			}
			return nil
		})
	}

	if LP_V2.View != nil {
		LP_V2.View.Subtitle = fmt.Sprintf("NPos:%d %s", len(list),
			cmn.FmtFloat64D(total_liq, true))
	}
}

func (p *LP_V2Pane) rebuidTemplate() string {
	w := cmn.CurrentWallet
	if w == nil {
		return "no open wallet"
	}

	if len(w.LP_V2_Positions) == 0 {
		return "(no positions)"
	}

	if len(lp_v2_info_list) == 0 {
		return "loading..."
	}

	temp := "Xch@Chain        Pair              Liq0      Liq1     Liq$ Address\n"

	for i, p := range lp_v2_info_list {

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
		temp += cmn.TagLink(
			fmt.Sprintf("%-16s", p.ProviderName),
			"open "+provider.URL,
			"open "+provider.URL)

		t0 := w.GetTokenByAddress(p.ChainId, p.Token0)
		t1 := w.GetTokenByAddress(p.ChainId, p.Token1)

		var pairLen int
		if t0 != nil && t1 != nil {
			pairStr := t0.Symbol + "/" + t1.Symbol
			pairLen = len(pairStr)
			temp += fmt.Sprintf(" %s", pairStr)
		} else {
			temp += " "
			if t0 != nil {
				temp += t0.Symbol
				pairLen = len(t0.Symbol)
			} else {
				temp += cmn.TagLink("???", "command token add "+b.Name+" "+p.Token0.String(), "Add token")
				pairLen = 3
			}

			temp += "/"
			pairLen++

			if t1 != nil {
				temp += t1.Symbol
				pairLen += len(t1.Symbol)
			} else {
				temp += cmn.TagLink("???", "command token add "+b.Name+" "+p.Token1.String(), "Add token")
				pairLen += 3
			}
		}
		// Pad to 13 chars
		if pairLen < 13 {
			temp += strings.Repeat(" ", 13-pairLen)
		}

		if t0 != nil {
			temp += cmn.TagFixedValueLink(p.Liquidity0, t0, 10)
		} else {
			temp += "          "
		}

		if t1 != nil {
			temp += cmn.TagFixedValueLink(p.Liquidity1, t1, 10)
		} else {
			temp += "          "
		}

		temp += cmn.TagFixedDollarLink(p.Liquidity0Dollars+p.Liquidity1Dollars, 10)

		temp += fmt.Sprintf(" %s", owner.Name)

		if i < len(lp_v2_info_list)-1 {
			temp += "\n"
		}

	}
	return temp
}
