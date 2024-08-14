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

type AppsPane struct {
	PaneDescriptor
}

var Apps AppsPane = AppsPane{
	PaneDescriptor: PaneDescriptor{
		MinWidth:  30,
		MinHeight: 1,
	},
}

var appsTemplate string

func (p *AppsPane) GetDesc() *PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *AppsPane) GetTemplate() string {
	return statusTemplate
}

func (p *AppsPane) SetView(x0, y0, x1, y1 int) {
	v, err := Gui.SetView("apps", x0, y0, x1, y1, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}

		log.Debug().Msg("SetView: apps")

		v.Title = "Apps"
		v.Autoscroll = false
		v.ScrollBar = true
		v.OnResize = func(v *gocui.View) {
			v.RenderTemplate(appsTemplate)
			v.ScrollTop()
		}
		v.OnOverHotspot = ProcessOnOverHotspot
		v.OnClickHotspot = ProcessOnClickHotspot
		p.rebuidTemplate()

	}
	p.PaneDescriptor.View = v
}

func AppsLoop() {
	ch := bus.Subscribe("wallet", "price")
	defer bus.Unsubscribe(ch)

	for msg := range ch {
		switch msg.Topic {
		case "wallet":
			switch msg.Type {
			case "open", "saved":
				Apps.rebuidTemplate()
			}
		case "price":
			switch msg.Type {
			case "updated":
				Apps.rebuidTemplate()
			}
		}
	}
}

func (p *AppsPane) rebuidTemplate() {
	temp := "<w>"

	if cmn.CurrentWallet != nil {
		w := cmn.CurrentWallet
		b := w.GetBlockchain(w.CurrentChain)
		a := w.GetAddress(w.CurrentAddress.String())

		if b != nil {
			t, err := w.GetNativeToken(b)
			if err != nil {
				log.Error().Err(err).Msg("rebuidTemplate: GetNativeToken")
				return
			}

			change := ""
			if t.PraceChange24 > 0 {
				change = fmt.Sprintf(" <color fg:green>\uf0d8%.2f%%</color>", t.PraceChange24)
			}
			if t.PraceChange24 < 0 {
				change = fmt.Sprintf(" <color fg:red>\uf0d7%.2f%%</color>", t.PraceChange24)
			}

			temp += fmt.Sprintf("<b>Chain:</b> %s | %s %s%s\n",
				b.Name, t.Symbol,
				TagShortDollarLink(t.Price),
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
						balance = fmt.Sprintf(" | %s", TagShortValueSymbolLink(blnc, t))

						if t.Price > 0 {
							dollars = fmt.Sprintf(" | %s", TagShortDollarValueLink(blnc, t))
						}
					}
				}
			}

			temp += fmt.Sprintf("<b> Addr:</b> %s %s%s%s\n",
				TagAddressShortLink(w.CurrentAddress), an, balance, dollars)
		}
	}

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
		temp += fmt.Sprintf("<b>   HW:</b> %s\n", cd)
	}

	statusTemplate = temp

	Gui.Update(func(g *gocui.Gui) error {
		if Status.View != nil {
			Status.View.RenderTemplate(statusTemplate)
		}
		return nil
	})
}
