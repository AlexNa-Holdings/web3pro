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
___       __    ______ _______________              
__ |     / /_______  /___|__  /__  __ \____________ 
__ | /| / /_  _ \_  __ \__/_ <__  /_/ /_  ___/  __ \
__ |/ |/ / /  __/  /_/ /___/ /_  ____/_  /   / /_/ /
____/|__/  \___//_.___//____/ /_/     /_/    \____/ `

func main() {
	var err error
	InitConfig()

	command.Init()

	ui.Gui, err = gocui.NewGui(gocui.OutputTrue, true)
	if err != nil {
		log.Fatal().Msgf("error creating gocui: %v", err)
	}
	defer ui.Gui.Close()

	ui.Gui.Mouse = true
	ui.Gui.Cursor = true
	ui.Gui.Highlight = true
	ui.Gui.SetManagerFunc(ui.Layout)

	if err := ui.Gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Fatal().Msgf("error setting keybinding: %v", err)
	}

	ui.SetTheme("dark") // temoprary

	ui.Is_ready_wg.Add(1)
	go func() {
		ui.Is_ready_wg.Wait()

		ui.Terminal.AutoCompleteFunc = command.AutoComplete
		ui.Terminal.ProcessCommandFunc = command.Process

		log.Trace().Msg("Started")

		ui.Printf("\n" + ui.F(ui.CurrentTheme.EmFgColor) + WEB3_PRO + ui.F(ui.Terminal.Screen.FgColor) + "\n\n")
		ui.Printf("by X:@AlexNa Telegram:@TheAlexNa\n")

		ui.Printf("Version: %s\n", ui.F(ui.CurrentTheme.EmFgColor)+VERSION+ui.F(ui.Terminal.Screen.FgColor))

		ui.Printf("Data folder: %s\n", ui.F(ui.CurrentTheme.EmFgColor)+DataFolder+ui.F(ui.Terminal.Screen.FgColor))
		ui.Printf("Log file: %s\n", ui.F(ui.CurrentTheme.EmFgColor)+LogPath+ui.F(ui.Terminal.Screen.FgColor))

		ui.Printf("\nType 'help' for help\n\n")

	}()

	if err := ui.Gui.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		log.Fatal().Msgf("error running gocui: %v", err)
	}
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
