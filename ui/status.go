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
	*gocui.View
	MinWidth  int
	MinHeight int
}

var Status *StatusPane = &StatusPane{
	MinWidth:  30,
	MinHeight: 4,
}

var statusTemplate = `Status`

func (p *StatusPane) SetView(g *gocui.Gui, x0, y0, x1, y1 int) {
	var err error

	if p.View, err = g.SetView("status", x0, y0, x1, y1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}
		p.View.Title = "Status"
		p.View.Autoscroll = false
		p.View.ScrollBar = true

		rebuidTemplate()

		p.View.OnResize = func(v *gocui.View) {
			v.RenderTemplate(statusTemplate)
			v.ScrollTop()
		}
	}
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
		temp += fmt.Sprintf("HW: %s\n", cd)
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
				change = fmt.Sprintf("<color fg:green>▲%2.f</color>", t.PraceChange24)
			}
			if t.PraceChange24 < 0 {
				change = fmt.Sprintf("<color fg:red>▼%2.f</color>", t.PraceChange24)
			}

			temp += fmt.Sprintf("Chain: %s | %s %s%s \n",
				b.Name, t.Symbol,
				cmn.FmtFloat64D(t.Price, false),
				change)
		}
	}

	statusTemplate = temp

	Gui.Update(func(g *gocui.Gui) error {

		Status.RenderTemplate(statusTemplate)
		return nil
	})
}
