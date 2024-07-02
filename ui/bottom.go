package ui

import (
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

type BottomPane struct {
	*gocui.View
	MinWidth  int
	MinHeight int
}

var Bottom *BottomPane = &BottomPane{
	MinWidth:  0,
	MinHeight: 1,
}

func (p *BottomPane) SetView(g *gocui.Gui) {
	var err error
	maxX, maxY := g.Size()

	if p.View, err = g.SetView("bottom", 0, maxY-2, maxX, maxY, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}
		p.View.Autoscroll = false
		p.View.Frame = false
	}
}

func (p *BottomPane) Printf(format string, args ...interface{}) {
	if p.View == nil {
		return
	}
	p.View.Clear()
	fmt.Fprintf(p.View, format, args...)
}
