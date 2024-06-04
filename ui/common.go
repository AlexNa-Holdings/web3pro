package ui

import (
	"sync"

	"github.com/AlexNa-Holdings/web3pro/gocui"
)

var Gui *gocui.Gui
var Is_ready = false
var Is_ready_wg sync.WaitGroup

func Layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	horizontal := maxX >= (Status.MinWidth + Confirm.MinWidth)

	FirstRowHeight := 0

	if horizontal {
		FirstRowHeight = max(Status.MinHeight, Confirm.MinHeight)
		Status.SetView(g, 0, 0, maxX/2-1, FirstRowHeight-1)
		Confirm.SetView(g, maxX/2, 0, maxX-1, FirstRowHeight-1)
	} else {
		FirstRowHeight = Status.MinHeight + Confirm.MinHeight
		Status.SetView(g, 0, 0, maxX-1, Status.MinHeight-1)
		Confirm.SetView(g, 0, Status.MinHeight, maxX-1, FirstRowHeight-1)
	}

	Terminal.SetView(g, 0, FirstRowHeight, maxX-1, maxY-2)

	Bottom.SetView(g)

	g.SetCurrentView("terminal.input")
	g.Cursor = true

	if !Is_ready {
		Is_ready_wg.Done()
		Is_ready = true
	}

	return nil
}
