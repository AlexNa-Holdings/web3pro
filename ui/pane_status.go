package ui

import (
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/eth"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

type StatusPane struct {
	PaneDescriptor
	On bool
}

var Status StatusPane = StatusPane{
	PaneDescriptor: PaneDescriptor{
		MinWidth:               45,
		MinHeight:              1,
		SupportCachedHightCalc: true,
	},
}

func (p *StatusPane) GetDesc() *PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *StatusPane) IsOn() bool {
	return p.On
}

func (p *StatusPane) SetOn(on bool) {
	p.On = on
}

func (p *StatusPane) EstimateLines(w int) int {
	return gocui.EstimateTemplateLines(p.GetTemplate(), w)
}

func (p *StatusPane) SetView(x0, y0, x1, y1 int, overlap byte) {
	v, err := Gui.SetView("status", x0, y0, x1, y1, overlap)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}
		p.PaneDescriptor.View = v
		v.JoinedFrame = true
		v.Title = "Status"
		v.Autoscroll = false
		v.ScrollBar = true
		v.OnResize = func(v *gocui.View) {
			v.RenderTemplate(p.GetTemplate())
			v.ScrollTop()
		}
		v.OnOverHotspot = ProcessOnOverHotspot
		v.OnClickHotspot = ProcessOnClickHotspot
		p.rebuidTemplate()
	}
}

func StatusLoop() {
	ch := bus.Subscribe("signer", "wallet", "price")
	defer bus.Unsubscribe(ch)

	for msg := range ch {
		switch msg.Topic {
		case "wallet":
			switch msg.Type {
			case "open", "saved":
				Status.rebuidTemplate()
			}
		case "signer":
			switch msg.Type {
			case "connected":
				Status.rebuidTemplate()
			}
		case "price":
			switch msg.Type {
			case "updated":
				Status.rebuidTemplate()
			}
		}
	}
}

func (p *StatusPane) rebuidTemplate() {
	temp := "<w>"

	if cmn.CurrentWallet != nil {
		w := cmn.CurrentWallet
		b := w.GetBlockchainByName(w.CurrentChain)
		a := w.GetAddress(w.CurrentAddress.String())

		if b != nil {
			t, err := w.GetNativeToken(b)
			if err != nil {
				log.Error().Err(err).Msg("rebuidTemplate: GetNativeToken")
				return
			}

			change := ""
			if t.PriceChange24 > 0 {
				change = fmt.Sprintf(" <color fg:green>\uf0d8%.2f%%</color>", t.PriceChange24)
			}
			if t.PriceChange24 < 0 {
				change = fmt.Sprintf(" <color fg:red>\uf0d7%.2f%%</color>", -t.PriceChange24)
			}

			temp += fmt.Sprintf("<b>  Chain:</b> %s | %s %s%s\n",
				b.Name, t.Symbol,
				cmn.TagShortDollarLink(t.Price),
				change)
		}

		if w.CurrentAddress != (common.Address{}) {
			an := ""
			if a != nil {
				an = a.Name
			}

			balance := ""
			dollars := ""

			if b != nil {
				blnc, err := eth.GetBalance(b, w.CurrentAddress)
				if err != nil {
					log.Error().Err(err).Msg("Status: GetBalance")
				} else {
					t, err := w.GetNativeToken(b)
					if err != nil {
						log.Error().Err(err).Msg("Status: GetNativeToken")
					} else {
						balance = cmn.TagShortValueSymbolLink(blnc, t)

						if t.Price > 0 {
							dollars = fmt.Sprintf(" %s", cmn.TagShortDollarValueLink(blnc, t))
						}
					}
				}
			}

			temp += fmt.Sprintf("<b>   Addr:</b> %s %s\n", cmn.TagAddressShortLink(w.CurrentAddress), an)
			temp += fmt.Sprintf("<b>Balance:</b> %s%s\n", balance, dollars)
		}
	}

	p.SetTemplate(temp)

	Gui.Update(func(g *gocui.Gui) error {
		if Status.View != nil {
			Status.View.RenderTemplate(p.GetTemplate())
		}
		return nil
	})
}
