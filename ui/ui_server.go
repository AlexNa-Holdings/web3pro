package ui

import (
	"sync"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

var Gui *gocui.Gui
var Is_ready = false
var Is_ready_wg sync.WaitGroup

func Init() {
	var err error

	Is_ready = false

	cmn.StandardOnClickHotspot = ProcessOnClickHotspot
	cmn.StandardOnOverHotspot = ProcessOnOverHotspot

	go Loop()
	go StatusLoop()
	go AppsLoop()

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

	Gui.OnPopupCloseGlobal = func() {
		Gui.SetCurrentView("terminal.input")
	}
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

			if hail.Timeout == 0 {
				hail.Timeout = cmn.Config.BusTimeout
			}
			if on_top := add(msg); on_top {
				HailPane.open(msg)
			}
		}
	case "remove-hail":
		if hail, ok := msg.Data.(*bus.B_Hail); ok {
			log.Trace().Msgf("ProcessHails: Remove hail received: %v", hail.Title)

			var m *bus.Message
			HQMutex.Lock()
			for i, h := range HailQueue {
				if h.Data == hail {
					m = HailQueue[i]
					break
				}
			}
			HQMutex.Unlock()

			if m != nil {
				remove(m)
			}
		}
	case "tick":
		if msg, ok := msg.Data.(*bus.B_TimerTick); ok {
			HQMutex.Lock()
			if ActiveRequest != nil {
				Gui.UpdateAsync(func(g *gocui.Gui) error {
					HailPane.UpdateSubtitle()
					return nil
				})

				hail := ActiveRequest.Data.(*bus.B_Hail)
				if hail.OnTick != nil {
					hail.OnTick(hail, msg.Tick)
				}
			}
			HQMutex.Unlock()
		}
	case "done":
		if d, ok := msg.Data.(*bus.B_TimerDone); ok {
			log.Trace().Msgf("Alert: %v", d.ID)
			if ActiveRequest != nil {
				if ActiveRequest.TimerID == d.ID {
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
				Apps.ShowPane()
			} else {
				Apps.HidePane()
			}
		}
	case "saved": // save wallet
		if cmn.CurrentWallet != nil {
			if cmn.CurrentWallet.AppsPaneOn {
				Apps.ShowPane()
			} else {
				Apps.HidePane()
			}
		}

	}
}
