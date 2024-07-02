package ui

import (
	"errors"
	"fmt"

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
	MinHeight: 3,
}

func (p *StatusPane) SetView(g *gocui.Gui, x0, y0, x1, y1 int) {
	var err error

	if p.View, err = g.SetView("status", x0, y0, x1, y1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}
		p.View.Title = "Status"
		p.View.Autoscroll = true

		fmt.Fprintln(p.View, "BlockChain: Ethereum")
		fmt.Fprintln(p.View, "Address: 0x1234567890")
	}
}
