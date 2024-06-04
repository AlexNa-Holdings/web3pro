package ui

import (
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/gocui"
)

type ConfirmPane struct {
	*gocui.View
	MinWidth  int
	MinHeight int
}

var Confirm *ConfirmPane = &ConfirmPane{
	MinWidth:  30,
	MinHeight: 9,
}

func (p *ConfirmPane) SetView(g *gocui.Gui, x0, y0, x1, y1 int) {
	var err error

	if p.View, err = g.SetView("confirm", x0, y0, x1, y1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			panic(err)
		}
		p.View.Title = "Confirm"
		p.View.Autoscroll = true
		p.View.SubTitleFgColor = Gui.ActionFgColor
		p.View.SubTitleBgColor = Gui.ActionBgColor
		fmt.Fprintln(p.View, "Nothing to confirm")

		p.Subtitle = "Confirmations (1)"
	}
}
