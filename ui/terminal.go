package ui

import (
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/gocui"
)

type TerminalPane struct {
	*gocui.View

	Screen             *gocui.View
	Input              *gocui.View
	Prefix             *gocui.View
	CommandPrefix      string
	FormattedPrefix    string
	ProcessCommandFunc func(string)
	History            []string
}

var Terminal *TerminalPane = &TerminalPane{
	CommandPrefix: "web3",
	History:       []string{},
}

func (p *TerminalPane) SetView(g *gocui.Gui, x0, y0, x1, y1 int) {
	var err error

	if p.View, err = g.SetView("terminal", x0, y0, x1, y1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			panic(err)
		}
		p.Title = "Terminal"
	}

	if p.Screen, err = g.SetView("terminal.screen", x0, y0, x1, y1-1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			panic(err)
		}
		p.Screen.Autoscroll = true
		p.Screen.Wrap = true
		p.Screen.Frame = false
	}

	prefix_len := len(p.CommandPrefix) + 1

	if y1-2 > y0 {

		if p.Input, err = g.SetView("terminal.input", prefix_len, y1-2, x1, y1, 0); err != nil {
			if !errors.Is(err, gocui.ErrUnknownView) {
				panic(err)
			}
			p.Input.Frame = false
			p.Input.Editable = true
			p.Input.Highlight = false
			p.Input.Editor = gocui.EditorFunc(terminalEditor)
		}

		if p.Prefix, err = g.SetView("terminal.prefix", x0, y1-2, prefix_len+1, y1, 0); err != nil {
			if !errors.Is(err, gocui.ErrUnknownView) {
				panic(err)
			}
			p.Prefix.Frame = false

			p.FormattedPrefix = FB(g.ActionFgColor, g.ActionBgColor) +
				p.CommandPrefix +
				FB(g.ActionBgColor, p.Prefix.BgColor) +
				"\ue0b0" +
				FB(p.Prefix.FgColor, p.Prefix.BgColor)

			fmt.Fprint(p.Prefix, p.FormattedPrefix)
			g.SetViewOnTop("terminal.prefix")
		}
	}
}

func terminalEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	switch key {
	case gocui.KeyEnter:
		fmt.Fprintln(Terminal.Screen, Terminal.FormattedPrefix+v.Buffer())
		if len(Terminal.History) > 100 {
			Terminal.History = Terminal.History[1:]
		}

		if len(v.Buffer()) > 0 &&
			(len(Terminal.History) == 0 ||
				(len(Terminal.History) > 0 &&
					Terminal.History[len(Terminal.History)-1] != v.Buffer())) {
			Terminal.History = append(Terminal.History, v.Buffer())
		}

		Terminal.ProcessCommandFunc(v.Buffer())
		Terminal.Input.Clear()
	case gocui.KeyArrowUp:
	case gocui.KeyArrowDown:

	default:
		gocui.DefaultEditor.Edit(v, key, ch, mod)
	}
}

func PrintErrorf(format string, a ...interface{}) {

	str := fmt.Sprintf(format, a...)

	fmt.Fprint(Terminal.Screen,
		F(CurrentTheme.ErrorFgColor)+
			str+
			F(Terminal.Screen.FgColor))
}

func Printf(format string, a ...interface{}) {

	str := fmt.Sprintf(format, a...)

	fmt.Fprint(Terminal.Screen, str)
}

func ResetColors() {
	fmt.Fprint(Terminal.Screen, FB(Terminal.Screen.FgColor, Terminal.Screen.BgColor))
}
