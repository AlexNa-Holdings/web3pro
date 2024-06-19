package ui

import (
	"strings"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/atotto/clipboard"
)

var Gui *gocui.Gui
var Is_ready = false
var Is_ready_wg sync.WaitGroup

func Init() {
	var err error

	Is_ready = false

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
	Notification.SetView(g)

	g.Cursor = true

	if !Is_ready {
		Is_ready_wg.Done()
		Is_ready = true
	}

	return nil
}

func ProcessClickHotspot(hs *gocui.Hotspot) {
	index := strings.Index(hs.Value, " ")

	if index == -1 {
		return
	}

	command := hs.Value[:index]
	param := hs.Value[index+1:]

	switch command {
	case "copy":
		clipboard.WriteAll(param)
		Notification.Show("Copied: " + param)
	case "command":
		Terminal.ProcessCommandFunc(param)
	}
}
