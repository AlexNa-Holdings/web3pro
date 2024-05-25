// SPDX-License-Identifier: Unlicense OR MIT

package main

// A Gio program that demonstrates Gio widgets. See https://gioui.org for more information.

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/AlexNa-Holdings/web3pro/ui"
)

func main() {
	ui.Init()

	go func() {
		w := new(app.Window)
		w.Option(app.Size(unit.Dp(800), unit.Dp(640)))
		w.Option(app.Title("Web3 Pro"))
		w.Option(app.Decorated(true))
		w.Option(app.CustomRenderer(true))

		if err := loop(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func loop(w *app.Window) error {
	events := make(chan event.Event)
	acks := make(chan struct{})

	go func() {
		for {
			ev := w.Event()
			events <- ev
			<-acks
			if _, ok := ev.(app.DestroyEvent); ok {
				return
			}
		}
	}()

	var ops op.Ops
	for e := range events {
		switch e := e.(type) {
		case app.DestroyEvent:
			acks <- struct{}{}
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			ui.MainPage(gtx)
			e.Frame(gtx.Ops)
		}
		acks <- struct{}{}
	}
	return nil
}
