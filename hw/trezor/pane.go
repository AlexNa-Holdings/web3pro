package trezor

import (
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/rs/zerolog/log"
)

var TP_Id int

type TrezorPane struct {
	ui.PaneDescriptor
	*Trezor
	Template string
	On       bool
	Id       int
	ViewName string
}

var Status TrezorPane = TrezorPane{
	PaneDescriptor: ui.PaneDescriptor{
		MinWidth:  45,
		MinHeight: 1,
	},
}

func NewTrezorPane(t *Trezor) *TrezorPane {
	TP_Id++
	p := &TrezorPane{
		PaneDescriptor: ui.PaneDescriptor{
			MinWidth:  30,
			MinHeight: 0,
		},
		Id:       TP_Id,
		ViewName: fmt.Sprintf("trezor-%d", TP_Id),
		Trezor:   t,
		On:       true,
	}

	return p
}

func (p *TrezorPane) GetDesc() *ui.PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *TrezorPane) IsOn() bool {
	return p.On
}

func (p *TrezorPane) SetOn(on bool) {
	p.On = on
}

func (p *TrezorPane) EstimateLines(w int) int {
	return gocui.EstimateTemplateLines(p.Template, w)
}

func (p *TrezorPane) SetView(x0, y0, x1, y1 int, overlap byte) {

	v, err := ui.Gui.SetView(p.ViewName, x0, y0, x1, y1, overlap)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}
		p.PaneDescriptor.View = v
		v.JoinedFrame = true
		v.Title = "HW Trezor"
		v.Subtitle = p.Name
		v.Autoscroll = false
		v.ScrollBar = true
		v.OnResize = func(v *gocui.View) {
			v.RenderTemplate(p.Template)
			v.ScrollTop()
		}
		// v.OnOverHotspot = ProcessOnOverHotspot
		// v.OnClickHotspot = ProcessOnClickHotspot
		p.rebuidTemplate()
	}
}

func (p *TrezorPane) rebuidTemplate() {
	temp := "<w>Trezor"

	p.Template = temp
}
