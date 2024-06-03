// Copyright 2014 The gocui Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"log"

	"github.com/AlexNa-Holdings/web3pro/command"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
)

const WEB3_PRO = `
/ \  /|/  __//  _ \\__  \  /  __\/  __\/  _ \
| |  |||  \  | | //  /  |  |  \/||  \/|| / \|
| |/\|||  /_ | |_\\ _\  |  |  __/|    /| \_/|
\_/  \|\____\\____//____/  \_/   \_/\_\\____/`

func main() {
	g, err := gocui.NewGui(gocui.OutputTrue, true)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	ui.SetTheme(g, "dark")

	g.Mouse = true
	g.Cursor = true
	g.Highlight = true
	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		log.Panicln(err)
	}
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	horizontal := maxX >= (ui.Status.MinWidth + ui.Confirm.MinWidth)

	FirstRowHeight := 0

	if horizontal {
		FirstRowHeight = max(ui.Status.MinHeight, ui.Confirm.MinHeight)
		ui.Status.SetView(g, 0, 0, maxX/2-1, FirstRowHeight-1)
		ui.Confirm.SetView(g, maxX/2, 0, maxX-1, FirstRowHeight-1)
	} else {
		FirstRowHeight = ui.Status.MinHeight + ui.Confirm.MinHeight
		ui.Status.SetView(g, 0, 0, maxX-1, ui.Status.MinHeight-1)
		ui.Confirm.SetView(g, 0, ui.Status.MinHeight, maxX-1, FirstRowHeight-1)
	}

	ui.Terminal.SetView(g, 0, FirstRowHeight, maxX-1, maxY-2)
	ui.Terminal.ProcessCommandFunc = command.Process
	ui.Terminal.AutoCompleteFunc = command.AutoComplete

	ui.Bottom.SetView(g)

	g.SetCurrentView("terminal.input")
	g.Cursor = true

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
