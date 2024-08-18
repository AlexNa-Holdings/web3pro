// Copyright 2014 The gocui Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/command"
	"github.com/AlexNa-Holdings/web3pro/eth"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/hw"
	"github.com/AlexNa-Holdings/web3pro/price"
	"github.com/AlexNa-Holdings/web3pro/sound"
	"github.com/AlexNa-Holdings/web3pro/sw"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/usb"
	"github.com/AlexNa-Holdings/web3pro/ws"
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

	bus.Init()
	sound.Init()
	sw.Init() // software wallets
	hw.Init() // hardware wallets
	usb.Init()
	eth.Init()
	command.Init()
	ws.Init()
	ui.Init()
	price.Init()

	defer ui.Gui.Close()

	ui.Is_ready_wg.Add(1)
	go func() {
		ui.Is_ready_wg.Wait()

		ui.Terminal.AutoCompleteFunc = command.AutoComplete

		ui.Printf(ui.F(ui.Theme.EmFgColor) + WEB3_PRO + ui.F(ui.Terminal.Screen.FgColor) + "\n\n")
		ui.Printf("by X:")
		ui.Terminal.Screen.AddLink("@AlexNa", "open https://x.com/AlexNa", "x.com/AlexNa", "")
		ui.Printf(" Telegram:")
		ui.Terminal.Screen.AddLink("@TheAlexNa", "open https://t.me/TheAlexNa", "t.me/TheAlexNa", "")

		ui.Printf("\nVersion: %s ", cmn.VERSION)
		ui.Terminal.Screen.AddLink("GitHub", "open https://github.com/AlexNa-Holdings/web3pro", "github.com/AlexNa-Holdings/web3pro", "")
		ui.Printf("\n")

		ui.Printf("\nData folder: ")
		ui.Terminal.Screen.AddLink(cmn.DataFolder, "copy "+cmn.DataFolder, "Copy data folder path to clipboard", "")
		ui.Printf("\n")

		ui.Printf("Log file: ")
		ui.Terminal.Screen.AddLink(cmn.LogPath, "copy "+cmn.LogPath, "Copy log file path to clipboard", "")
		ui.Printf("\n")

		ui.Printf("Config file: ")
		ui.Terminal.Screen.AddLink(cmn.ConfPath, "copy "+cmn.ConfPath, "Copy config file path to clipboard", "")
		ui.Printf("\n")

		ui.Printf("\nType 'help' for help\n\n")
	}()

	if err := ui.Gui.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		log.Fatal().Msgf("error running gocui: %v", err)
	}

	cmn.SaveConfig()
}
