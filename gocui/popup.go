package gocui

import (
	"errors"
)

type Popup struct {
	*View
	Width, Height   int
	Title, Subtitle string
}

func (g *Gui) ShowPopup(p *Popup) {
	g.popup = p
}

func (p *Popup) Layout(g *Gui) error {

	maxX, maxY := g.Size()

	// Center the popup

	if v, err := g.SetView("popup`", maxX/2-p.Width/2, maxY/2-p.Height/2, maxX/2+p.Width/2, maxY/2+p.Height/2, 0); err != nil {
		if !errors.Is(err, ErrUnknownView) {
			return err
		}

		v.Frame = true
		v.Title = p.Title
		v.Subtitle = p.Subtitle
		g.popup.View = v
	}

	return nil
}
