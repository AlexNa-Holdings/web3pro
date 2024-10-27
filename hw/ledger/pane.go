package ledger

import (
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/rs/zerolog/log"
)

var TP_Id int

type LedgerPane struct {
	ui.PaneDescriptor
	*Ledger
	On       bool
	Id       int
	ViewName string
	Mode     string
	Template string
}

func NewLedgerPane(t *Ledger) *LedgerPane {
	TP_Id++
	p := &LedgerPane{
		PaneDescriptor: ui.PaneDescriptor{
			MinWidth:  30,
			MinHeight: 0,
		},
		Id:       TP_Id,
		ViewName: fmt.Sprintf("Ledger-%d", TP_Id),
		Ledger:   t,
		On:       true,
	}

	go p.loop()

	return p
}

func (p *LedgerPane) loop() {

	ch := bus.Subscribe(p.ViewName, "timer")
	for msg := range ch {
		if msg.RespondTo != 0 {
			continue // ignore responses
		}
		switch msg.Type {
		}
	}
}

func (p *LedgerPane) GetDesc() *ui.PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *LedgerPane) IsOn() bool {
	return p.On
}

func (p *LedgerPane) SetOn(on bool) {
	p.On = on
}

func (p *LedgerPane) EstimateLines(w int) int {
	return gocui.EstimateTemplateLines(p.GetTemplate(), w)
}

func (p *LedgerPane) SetView(x0, y0, x1, y1 int, overlap byte) {

	v, err := ui.Gui.SetView(p.ViewName, x0, y0, x1, y1, overlap)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}
		p.PaneDescriptor.View = v
		v.JoinedFrame = true
		v.Title = "HW Ledger"
		v.Subtitle = p.Name
		v.Autoscroll = false
		v.ScrollBar = true
		v.OnResize = func(v *gocui.View) {
			v.RenderTemplate(p.GetTemplate())
			v.ScrollTop()
		}
		v.OnOverHotspot = cmn.StandardOnOverHotspot
		v.OnClickHotspot = func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				param := cmn.Split3(hs.Value)

				switch param[0] {
				case "button":
					switch param[1] {
					case "get_name":
						if p.Ledger != nil {
							n, err := getName(p.USB_ID)
							if err != nil {
								log.Error().Err(err).Msg("Error initializing ledger")
								return
							}

							p.Ledger.Name = n
							p.rebuidTemplate()
						}
					}
				}
				cmn.StandardOnClickHotspot(v, hs)
			}
		}
		p.rebuidTemplate()
	}
}

func (p *LedgerPane) SetMode(mode string) {
	p.Mode = mode

	if p.View != nil {
		if mode != "" {
			p.View.EmFgColor = ui.Gui.ActionBgColor
		} else {
			p.View.EmFgColor = ui.Gui.HelpFgColor
		}
	}

	p.rebuidTemplate()
}

func (p *LedgerPane) rebuidTemplate() {
	temp := ""
	switch p.Mode {
	case "template":
		temp = p.GetTemplate()
	default:
		if p.Ledger != nil {
			if p.Name != "" {
				temp += fmt.Sprintf("<w><b>Product:<b> %s", p.Ledger.Product)
			} else {
				temp += "<c>\n<button id:get_name text:'Request Devive Name'>\n"
			}
		}
	}

	p.SetTemplate(temp)

	if p.View != nil {
		p.View.Subtitle = p.Name
	}

	ui.Gui.Update(func(g *gocui.Gui) error {
		if p.View != nil {
			p.View.RenderTemplate(p.GetTemplate())
		}
		return nil
	})
}
