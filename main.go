// Copyright 2014 The gocui Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"

	"github.com/AlexNa-Holdings/web3pro/cmn"
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
	cmn.InitConfig()

	command.Init()
	ui.Init()
	defer ui.Gui.Close()

	ui.Is_ready_wg.Add(1)
	go func() {
		ui.Is_ready_wg.Wait()

		ui.Terminal.AutoCompleteFunc = command.AutoComplete
		ui.Terminal.ProcessCommandFunc = command.Process

		ui.Printf("\n" + ui.F(ui.Theme.EmFgColor) + WEB3_PRO + ui.F(ui.Terminal.Screen.FgColor) + "\n\n")
		ui.Printf("by X:@AlexNa Telegram:@TheAlexNa\n")

		ui.Printf("Version: %s\n", cmn.VERSION)

		ui.Printf("Data folder: ")
		ui.Terminal.Screen.AddLink(cmn.DataFolder, "copy "+cmn.DataFolder, "Copy data folder path to clipboard")
		ui.Printf("\n")

		ui.Printf("Log file: ")
		ui.Terminal.Screen.AddLink(cmn.LogPath, "copy "+cmn.LogPath, "Copy log file path to clipboard")
		ui.Printf("\n")

		ui.Printf("Config file: ")
		ui.Terminal.Screen.AddLink(cmn.ConfPath, "copy "+cmn.ConfPath, "Copy config file path to clipboard")
		ui.Printf("\n")

		ui.Printf("\nType 'help' for help\n\n")
	}()

	if err := ui.Gui.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		log.Fatal().Msgf("error running gocui: %v", err)
	}

	cmn.SaveConfig()
}
