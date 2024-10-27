package ui

import (
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

type AppPane struct {
	PaneDescriptor
	On bool
}

var App AppPane = AppPane{
	PaneDescriptor: PaneDescriptor{
		MinWidth:               45,
		MinHeight:              1,
		MaxHeight:              20,
		SupportCachedHightCalc: true,
	},
}

func (p *AppPane) GetDesc() *PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *AppPane) EstimateLines(w int) int {

	return gocui.EstimateTemplateLines(p.GetTemplate(), w)
}

func (p *AppPane) IsOn() bool {
	return p.On
}

func (p *AppPane) SetOn(on bool) {
	p.On = on
}

func (p *AppPane) SetView(x0, y0, x1, y1 int, overlap byte) {
	v, err := Gui.SetView("app", x0, y0, x1, y1, overlap)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}

		p.PaneDescriptor.View = v
		v.JoinedFrame = true
		v.Title = "Web Application"
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

func AppsLoop() {
	ch := bus.Subscribe("wallet", "price")
	defer bus.Unsubscribe(ch)

	for msg := range ch {
		switch msg.Topic {
		case "wallet":
			switch msg.Type {
			case "open", "saved":
				App.SetTemplate(App.rebuidTemplate())
				Gui.Update(func(g *gocui.Gui) error {
					if App.View != nil {
						App.View.RenderTemplate(App.GetTemplate())
					}
					return nil
				})
			}
		case "price":
			switch msg.Type {
			case "updated":
				App.SetTemplate(App.rebuidTemplate())
				Gui.Update(func(g *gocui.Gui) error {
					if App.View != nil {
						App.View.RenderTemplate(App.GetTemplate())
					}
					return nil
				})
			}
		}
	}
}

func (p *AppPane) rebuidTemplate() string {
	temp := "<w>"

	w := cmn.CurrentWallet
	if w == nil {
		return temp + "No wallet selected"
	}

	o := w.GetOrigin(w.CurrentOrigin)
	if o == nil {
		return temp + "No origin selected"
	}

	temp += "<b>  App: </b>"

	temp += cmn.TagLink(o.ShortName(),
		"start_command app set ",
		"Set the app")

	temp += "\n<b>  URL: </b>"

	temp += cmn.TagLink(o.URL,
		"start_command app set ",
		"Set the app")

	temp += "\n<b>Chain: </b>"
	b := w.GetBlockchain(o.ChainId)
	bname := "unknown"
	if b != nil {
		bname = b.Name
	}
	bname = fmt.Sprintf("%s (%d)", bname, o.ChainId)
	temp += cmn.TagLink(
		bname,
		"start_command app chain '"+o.URL+"' ",
		"Change chain")

	temp += "\n<b> Addr: </b>"
	for i, na := range o.Addresses {

		name := "unknown"
		a := w.GetAddress(na.String())
		if a != nil {
			name = a.Name
		}

		if i == 0 {
			temp += cmn.TagAddressShortLink(na)
			temp += " " + cmn.TagLink(cmn.ICON_DELETE,
				"command app remove_addr '"+o.URL+"' '"+na.String()+"'",
				"Remove access for the address")
			temp += "  " + name
			temp += "\n"
		} else {
			temp += "       " + cmn.TagAddressShortLink(na)
			temp += " " + cmn.TagLink(cmn.ICON_DELETE,
				"command app remove_addr '"+o.URL+"' '"+na.String()+"'",
				"Remove access for the address")
			temp += cmn.TagLink(cmn.ICON_PROMOTE,
				"command app promote_addr '"+o.URL+"' '"+na.String()+"'",
				"Promote address")
			temp += name
			temp += "\n"
		}
	}

	temp += "       " + cmn.TagLink(cmn.ICON_ADD,
		"start_command app add_addr '"+o.URL+"' ",
		"Add address")

	return temp
}
