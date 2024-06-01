package ui

import (
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/gocui"
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

	if p.View, err = g.SetView("Bottom", 0, maxY-2, maxX-1, maxY-1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			panic(err)
		}
		p.View.Title = "Terminal"
		p.View.Autoscroll = true
		p.View.Frame = false
		fmt.Fprintln(p.View, "View with default frame color")
		fmt.Fprintln(p.View, "It's connected to v4 with overlay RIGHT.")
	}
}
