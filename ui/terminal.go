package ui

import (
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/gocui"
)

type TerminalPane struct {
	*gocui.View
}

var Terminal *TerminalPane = &TerminalPane{}

func (p *TerminalPane) SetView(g *gocui.Gui, x0, y0, x1, y1 int) {
	var err error

	if p.View, err = g.SetView("terminal", x0, y0, x1, y1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			panic(err)
		}
		p.Title = "Terminal"
		p.Autoscroll = true
		p.Editable = true
		p.Wrap = true
		p.Overwrite = false

		fmt.Fprintln(p.View, "View with default frame color")
		fmt.Fprintln(p.View, "It's connected to v4 with overlay RIGHT.")
	}
}
