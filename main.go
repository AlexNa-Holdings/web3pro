// SPDX-License-Identifier: Unlicense OR MIT

package main

// A Gio program that demonstrates Gio widgets. See https://gioui.org for more information.

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/ui/pages/main_page"
	"github.com/AlexNa-Holdings/web3pro/ui/pages/settings"
)

func main() {
	ui.Init()

	go func() {
		w := new(app.Window)
		w.Option(app.Size(unit.Dp(800), unit.Dp(640)))
		w.Option(app.Title("Web3 Pro"))
		w.Option(app.Decorated(false))
		w.Option(app.CustomRenderer(true))

		if err := loop(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func loop(w *app.Window) error {
	th := ui.UI.Theme.BasicTheme
	var ops op.Ops

	router := ui.NewRouter(w)
	router.Register(0, main_page.New(&router))
	router.Register(1, settings.New(&router))

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			router.Layout(gtx, th)
			e.Frame(gtx.Ops)
		}
	}
}
