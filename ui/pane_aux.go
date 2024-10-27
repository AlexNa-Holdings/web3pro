package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

var AuxP_Id int

type AuxPane struct {
	PaneDescriptor
	On       bool
	Id       int
	Title    string
	ViewName string
}

func NewAuxPane(title, template string) *AuxPane {
	AuxP_Id++
	p := &AuxPane{
		PaneDescriptor: PaneDescriptor{
			MinWidth:  30,
			MinHeight: 0,
		},
		Id:       AuxP_Id,
		ViewName: fmt.Sprintf("aux-%d", AuxP_Id),
		On:       true,
		Title:    title,
	}

	p.SetTemplate(template)

	return p
}

func (p *AuxPane) GetDesc() *PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *AuxPane) IsOn() bool {
	return p.On
}

func (p *AuxPane) SetOn(on bool) {
	p.On = on
}

func (p *AuxPane) EstimateLines(w int) int {
	return gocui.EstimateTemplateLines(p.GetTemplate(), w)
}

func (p *AuxPane) SetView(x0, y0, x1, y1 int, overlap byte) {

	v, err := Gui.SetView(p.ViewName, x0, y0, x1, y1, overlap)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}
		p.PaneDescriptor.View = v
		v.JoinedFrame = true
		v.Title = p.Title
		v.Subtitle = ""
		v.Autoscroll = false
		v.ScrollBar = true
		v.OnResize = func(v *gocui.View) {
			v.RenderTemplate(p.GetTemplate())
			v.ScrollTop()
		}

		v.SubTitleFgColor = Theme.HelpFgColor
		v.SubTitleBgColor = Theme.HelpBgColor
		v.FrameColor = Gui.ActionBgColor
		v.TitleColor = Gui.ActionFgColor
		v.EmFgColor = Gui.ActionBgColor

		v.OnOverHotspot = cmn.StandardOnOverHotspot
		v.OnClickHotspot = func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				param := cmn.Split3(hs.Value)

				switch param[0] {
				case "button":
					switch strings.ToLower(param[1]) {
					case "cancel":
						p.SetOn(false)
						return
					}
				}
			}
			cmn.StandardOnClickHotspot(v, hs)
		}

		v.RenderTemplate(p.GetTemplate())
	}
}
