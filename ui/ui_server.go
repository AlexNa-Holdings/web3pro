package ui

import (
	"fmt"
	"sync"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

var Gui *gocui.Gui
var Is_ready = false
var Is_ready_wg sync.WaitGroup

var RUNES = []rune{'─', '│', '┌', '┐', '└', '┘', '├', '┤', '┬', '┴', '┼'}

func Init() {
	var err error

	Is_ready = false

	cmn.StandardOnClickHotspot = ProcessOnClickHotspot
	cmn.StandardOnOverHotspot = ProcessOnOverHotspot

	go Loop()
	go StatusLoop()
	go AppsLoop()
	go LP_V3Loop()

	Gui, err = gocui.NewGui(gocui.OutputTrue, true)
	if err != nil {
		log.Fatal().Msgf("error creating gocui: %v", err)
	}
	Gui.Mouse = true
	Gui.Cursor = true
	Gui.Highlight = true
	SetTheme(cmn.Config.Theme)
	Gui.SetManagerFunc(Layout)
	SetKeybindings()
}

func SetKeybindings() error {
	if err := Gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Fatal().Msgf("error setting keybinding: %v", err)
	}

	if err := Gui.SetKeybinding("terminal.autocomplete", gocui.MouseLeft, gocui.ModNone, OnAutocompleteMouseDown); err != nil {
		log.Fatal().Msgf("error setting keybinding: %v", err)
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	log.Info().Msg("Quitting")
	return gocui.ErrQuit
}

func Loop() {
	ch := bus.Subscribe("ui", "timer", "wallet")
	defer bus.Unsubscribe(ch)

	for msg := range ch {
		go process(msg)
	}
}

func process(msg *bus.Message) {
	switch msg.Type {
	case "hail":
		log.Trace().Msg("ProcessHails: Hail received")
		if hail, ok := msg.Data.(*bus.B_Hail); ok {
			log.Trace().Msgf("Hail received: %s", hail.Title)

			if on_top := add(msg); on_top {
				HailPane.open(msg)
			}
		}
	case "remove-hail":
		if m, ok := msg.Data.(*bus.Message); ok {
			if m != nil {
				remove(m)
			}
		}
	case "popup":
		processPopup(msg)
	case "start_command":
		if text, ok := msg.Data.(string); ok {
			Terminal.Input.Clear()
			fmt.Fprint(Terminal.Input, text)
			Terminal.Input.SetCursor(len(text), 0)
			// try autocomplete again
			if Terminal.AutoCompleteFunc != nil {
				t, o, h := Terminal.AutoCompleteFunc(text)
				Terminal.ShowAutocomplete(t, o, h)
			}
		}
	case "tick":
		if msg, ok := msg.Data.(*bus.B_TimerTick); ok {
			if ActiveRequest != nil {
				Gui.UpdateAsync(func(g *gocui.Gui) error {
					HailPane.UpdateSubtitle(msg.Left)
					return nil
				})

				hail := ActiveRequest.Data.(*bus.B_Hail)
				if hail == nil {
					log.Error().Msg("ActiveRequest.Data is not of type HailRequest (SHOULD NEVER HAPPEN)")
					return
				}

				if hail.OnTick != nil {
					hail.OnTick(ActiveRequest, msg.Tick)
				}
			}
		}
	case "done":
		if id, ok := msg.Data.(int); ok {
			if ActiveRequest != nil {
				if ActiveRequest.TimerID == id {
					cancel(ActiveRequest)
				}
			}
		}
	case "notify":
		if text, ok := msg.Data.(string); ok {
			Notification.ShowEx(text, false)
		}
	case "notify-error":
		if text, ok := msg.Data.(string); ok {
			Notification.ShowEx(text, true)
		}
	case "open": // open wallet
		Status.ShowPane()
		if cmn.CurrentWallet != nil {
			if cmn.CurrentWallet.AppsPaneOn {
				App.ShowPane()
			} else {
				App.HidePane()
			}

			if cmn.CurrentWallet.LP_V3PaneOn {
				LP_V3.ShowPane()
			} else {
				LP_V3.HidePane()
			}
		}
	case "saved": // save wallet
		if cmn.CurrentWallet != nil {
			if cmn.CurrentWallet.AppsPaneOn {
				App.ShowPane()
			} else {
				App.HidePane()
			}
		}
	}
}
