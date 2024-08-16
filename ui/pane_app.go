package ui

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

type AppPane struct {
	PaneDescriptor
	Template string
}

var App AppPane = AppPane{
	PaneDescriptor: PaneDescriptor{
		MinWidth:  30,
		MinHeight: 1,
		MaxHeight: 20,
	},
}

func (p *AppPane) GetDesc() *PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *AppPane) GetTemplate() string {
	return p.Template
}

func (p *AppPane) SetView(x0, y0, x1, y1 int) {
	v, err := Gui.SetView("app", x0, y0, x1, y1, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}

		log.Debug().Msg("SetView: apps")

		v.Title = "Application"
		v.ScrollBar = true
		v.OnResize = func(v *gocui.View) {
			v.RenderTemplate(p.Template)
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
				App.Template = App.rebuidTemplate()
				Gui.Update(func(g *gocui.Gui) error {
					if App.View != nil {
						App.View.RenderTemplate(App.Template)
					}
					return nil
				})
			}
		case "price":
			switch msg.Type {
			case "updated":
				App.Template = App.rebuidTemplate()
				Gui.Update(func(g *gocui.Gui) error {
					if App.View != nil {
						App.View.RenderTemplate(App.Template)
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

	temp += `<b>  App: </b>`
	// Parse the URL
	parsedURL, err := url.Parse(o.URL)
	if err != nil {
		temp += cmn.TagLink(o.URL,
			"start_command app set '"+o.URL+"'",
			"Set the app")
	} else {
		temp += cmn.TagLink(parsedURL.Hostname(),
			"start_command app set '"+o.URL+"'",
			"Set the app")
	}

	temp += "\n<b>Chain: </b>"
	b := w.GetBlockchainById(o.ChainId)
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
			temp += " " + cmn.TagLink(gocui.ICON_DELETE,
				"command app remove_addr '"+o.URL+"' '"+na.String()+"'",
				"Remove access for the address")
			temp += "  " + name
			temp += "\n"
		} else {
			temp += "       " + cmn.TagAddressShortLink(na)
			temp += " " + cmn.TagLink(gocui.ICON_DELETE,
				"command app remove_addr '"+o.URL+"' '"+na.String()+"'",
				"Remove access for the address")
			temp += cmn.TagLink(gocui.ICON_PROMOTE,
				"command app promote_addr '"+o.URL+"' '"+na.String()+"'",
				"Promote address")
			temp += name
			temp += "\n"
		}
	}

	temp += "       " + cmn.TagLink(gocui.ICON_ADD,
		"start_command app add_addr '"+o.URL+"' ",
		"Add address")

	return temp
}
