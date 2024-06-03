// Copyright 2014 The gocui Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"

	"github.com/AlexNa-Holdings/web3pro/command"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/rs/zerolog/log"
)

const WEB3_PRO = `
___       __    ______ ________ ________              
__ |     / /_______  /___|__  / ___  __ \____________ 
__ | /| / /_  _ \_  __ \__/_ <  __  /_/ /_  ___/  __ \
__ |/ |/ / /  __/  /_/ /___/ /  _  ____/_  /   / /_/ /
____/|__/  \___//_.___//____/   /_/     /_/    \____/ `

func main() {
	InitConfig()

	command.Init()

	g, err := gocui.NewGui(gocui.OutputTrue, true)
	if err != nil {
		log.Fatal().Msgf("error creating gocui: %v", err)
	}
	defer g.Close()

	ui.SetTheme(g, "dark")

	g.Mouse = true
	g.Cursor = true
	g.Highlight = true
	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Fatal().Msgf("error setting keybinding: %v", err)
	}

	ui.Is_ready_wg.Add(1)
	go func() {
		ui.Is_ready_wg.Wait()
		ui.Printf(ui.F(ui.CurrentTheme.EmFgColor) + WEB3_PRO + ui.F(ui.Terminal.Screen.FgColor) + "\n\n")
		ui.Printf("by X:@AlexNa Telegram:@TheAlexNa\n")

		ui.Printf("Version: %s\n", ui.F(ui.CurrentTheme.EmFgColor)+VERSION+ui.F(ui.Terminal.Screen.FgColor))

		ui.Printf("Data folder: %s\n", ui.F(ui.CurrentTheme.EmFgColor)+DataFolder+ui.F(ui.Terminal.Screen.FgColor))
		ui.Printf("Log file: %s\n", ui.F(ui.CurrentTheme.EmFgColor)+LogPath+ui.F(ui.Terminal.Screen.FgColor))

		log.Trace().Msg("Started")

		ui.Printf("\nType 'help' for help\n\n")

	}()

	if err := g.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		log.Fatal().Msgf("error running gocui: %v", err)
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

	if !ui.Is_ready {
		ui.Is_ready_wg.Done()
		ui.Is_ready = true
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
