package ui

import (
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"

	"github.com/AlexNa-Holdings/web3pro/bus"
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

	cmn.StandardOnClickHotspot = ProcessOnClickHotspot
	cmn.StandardOnOverHotspot = ProcessOnOverHotspot

	go Loop()

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

func Layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	FirstRowHeight := 0

	if HailPane.View != nil {
		HailPane.MinHeight = max(HailPane.MinHeight, HailPane.View.LinesHeight()+2)
		HailPane.MinHeight = min(HailPane.MinHeight, maxY-10)

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

func ProcessOnClickHotspot(v *gocui.View, hs *gocui.Hotspot) {
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
		bus.Send("ui", "command", param)
	case "open":
		cmn.OpenBrowser(param)
	}
}

func ProcessOnOverHotspot(v *gocui.View, hs *gocui.Hotspot) {
	if hs != nil {
		Bottom.Printf(hs.Tip)
	} else {
		Bottom.Printf("")
	}
}

func AddAddressLink(v *gocui.View, a common.Address) {
	if v == nil {
		v = Terminal.Screen
	}

	v.AddLink(a.String(), "copy "+a.String(), "Copy address", "")
}

func AddAddressShortLink(v *gocui.View, a common.Address) {
	if v == nil {
		v = Terminal.Screen
	}

	s := a.String()
	v.AddLink(s[:6]+gocui.ICON_3DOTS+s[len(s)-4:], "copy "+a.String(), "Copy "+a.String(), "")
}

func AddValueLink(v *gocui.View, val *big.Int, t *cmn.Token) {
	if v == nil {
		return
	}

	if t == nil {
		return
	}

	text := cmn.FmtAmount(val, t.Decimals, true)
	n := cmn.Amount2Str(val, t.Decimals)

	v.AddLink(text, "copy "+n, "Copy "+n, "")
}

func AddValueSymbolLink(v *gocui.View, val *big.Int, t *cmn.Token) {
	if v == nil {
		return
	}

	if t == nil {
		return
	}

	text := cmn.FmtAmount(val, t.Decimals, true) + t.Symbol
	n := cmn.Amount2Str(val, t.Decimals)

	v.AddLink(text, "copy "+n, "Copy "+n, "")
}
