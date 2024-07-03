package ui

import (
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
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

	go ProcessHails()

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

	FirstRowHeight := 0

	if HailPane.View != nil {
		HailPane.MinHeight = max(5, HailPane.View.LinesHeight()+1)

		horizontal := maxX >= (Status.MinWidth + HailPane.MinWidth)

		if horizontal {
			HailPane.MinHeight = min(HailPane.MinHeight, maxY-10)
			FirstRowHeight = max(Status.MinHeight, HailPane.MinHeight)
			Status.SetView(g, 0, 0, maxX/2-1, FirstRowHeight-1)
			HailPane.SetView(g, maxX/2, 0, maxX-1, FirstRowHeight-1)
		} else {
			HailPane.MinHeight = min(HailPane.MinHeight, maxY-Status.MinHeight-10)
			FirstRowHeight = Status.MinHeight + HailPane.MinHeight
			Status.SetView(g, 0, 0, maxX-1, Status.MinHeight-1)
			HailPane.SetView(g, 0, Status.MinHeight, maxX-1, FirstRowHeight-1)
		}
	} else {
		FirstRowHeight = Status.MinHeight
		Status.SetView(g, 0, 0, maxX-1, FirstRowHeight-1)
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

func ProcessClicksOnScreen(hs *gocui.Hotspot) {
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

func AddAddressLink(v *gocui.View, a *common.Address) {
	if v == nil {
		v = Terminal.Screen
	}

	if a == nil {
		return
	}

	v.AddLink(a.String(), "copy "+a.String(), "Copy address", "")
}

func AddAddressShortLink(v *gocui.View, a *common.Address) {
	if v == nil {
		v = Terminal.Screen
	}

	if a == nil {
		return
	}

	s := a.String()
	v.AddLink(s[:6]+gocui.ICON_3DOTS+s[len(s)-4:], "copy "+a.String(), "Copy address", "")
}
