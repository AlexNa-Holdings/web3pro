package ui

import (
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

type StatusPane struct {
	PaneDescriptor
}

var Status StatusPane = StatusPane{
	PaneDescriptor: PaneDescriptor{
		MinWidth:  30,
		MinHeight: 1,
	},
}

var statusTemplate string

func (p *StatusPane) GetDesc() *PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *StatusPane) GetTemplate() string {
	return statusTemplate
}

func (p *StatusPane) SetView(x0, y0, x1, y1 int) {
	v, err := Gui.SetView("status", x0, y0, x1, y1, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}
		v.Title = "Status"
		v.Autoscroll = false
		v.ScrollBar = true
		v.OnResize = func(v *gocui.View) {
			v.RenderTemplate(statusTemplate)
			v.ScrollTop()
		}
		rebuidTemplate()

	}
	p.PaneDescriptor.View = v
}

func StatusLoop() {
	ch := bus.Subscribe("signer", "wallet", "price")
	defer bus.Unsubscribe(ch)

	for msg := range ch {
		switch msg.Topic {
		case "wallet":
			switch msg.Type {
			case "open":
				rebuidTemplate()
			}
		case "signer":
			switch msg.Type {
			case "connected":
				rebuidTemplate()
			}
		case "price":
			switch msg.Type {
			case "updated":
				rebuidTemplate()
			}
		}
	}
}

func rebuidTemplate() {
	temp := "<w>"

	// connected devices
	cd := ""
	r := bus.Fetch("signer", "list", &bus.B_SignerList{Type: "ledger"})
	if r.Error == nil {
		if res, ok := r.Data.(*bus.B_SignerList_Response); ok {
			for i, n := range res.Names {
				if i > 0 {
					cd += ", "
				}
				cd = fmt.Sprintf("%s(L)", n)
			}
		}
	}
	r = bus.Fetch("signer", "list", &bus.B_SignerList{Type: "trezor"})
	if r.Error == nil {
		if res, ok := r.Data.(*bus.B_SignerList_Response); ok {
			for i, n := range res.Names {
				if i > 0 {
					cd += ", "
				}
				cd = fmt.Sprintf("%s(T)", n)
			}
		}
	}
	if cd != "" {
		temp += fmt.Sprintf("<b>HW:</b> %s\n", cd)
	}

	if cmn.CurrentWallet != nil {
		w := cmn.CurrentWallet

		if w.CurrentChain != "" {
			b := w.GetBlockchain(w.CurrentChain)
			t, err := w.GetNativeToken(b)
			if err != nil {
				log.Error().Err(err).Msg("rebuidTemplate: GetNativeToken")
				return
			}

			change := ""
			if t.PraceChange24 > 0 {
				change = fmt.Sprintf("<color fg:green>\uf0d8%.2f%%</color>", t.PraceChange24)
			}
			if t.PraceChange24 < 0 {
				change = fmt.Sprintf("<color fg:red>\uf0d7%.2f%%</color>", t.PraceChange24)
			}

			temp += fmt.Sprintf("<b>Chain:</b> %s | %s %s%s\n",
				b.Name, t.Symbol,
				cmn.FmtFloat64D(t.Price, false),
				change)
		}
	}

	statusTemplate = temp

	Gui.Update(func(g *gocui.Gui) error {
		if Status.View == nil {
			Status.PaneDescriptor.ShowPane()
		} else {
			Status.View.RenderTemplate(statusTemplate)
		}
		return nil
	})
}
