package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/gocui"
)

type TerminalPane struct {
	*gocui.View

	Screen             *gocui.View
	Input              *gocui.View
	Prefix             *gocui.View
	AutoComplete       *gocui.View
	AutoCompleteOn     bool
	ACOptions          *[]ACOption // autocomplete options
	ACTitle            string      //autocomplete title
	CommandPrefix      string
	FormattedPrefix    string
	ProcessCommandFunc func(string)
	AutoCompleteFunc   func(string) (string, *[]ACOption, string)
	History            []string
	*gocui.Gui
}

type ACOption struct {
	Name   string
	Result string
}

var Terminal *TerminalPane = &TerminalPane{
	CommandPrefix:  "web3",
	History:        []string{},
	ACOptions:      &[]ACOption{},
	AutoCompleteOn: false,
}

func (p *TerminalPane) SetView(g *gocui.Gui, x0, y0, x1, y1 int) {
	var err error

	p.Gui = g

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

		if p.AutoCompleteOn {
			p.layoutAutocomplete(p.ACTitle, p.ACOptions, "")
		}
	}
}

func (t *TerminalPane) ShowAutocomplete(title string, options *[]ACOption, highlite string) {
	t.HideAutocomplete()

	if len(*options) > 0 {

		t.ACOptions = options
		t.ACTitle = title
		t.AutoCompleteOn = true
	}
}

func (p *TerminalPane) HideAutocomplete() {
	if p.AutoCompleteOn {
		p.Gui.DeleteView("terminal.autocomplete")
		p.AutoCompleteOn = false
	}
}

func (t *TerminalPane) layoutAutocomplete(title string, options *[]ACOption, highlite string) {
	var err error

	longest_option := 0
	for _, option := range *options {
		if len(option.Name) > longest_option {
			longest_option = len(option.Name)
		}
	}

	//calculate the frame size
	cursor_x, _ := t.Input.Cursor()
	ix0, _, ix1, _ := t.Input.Dimensions()
	_, sy0, sx1, sy1 := t.Screen.Dimensions()

	frame_width := max(longest_option+2, len(title)+16) // make the title visible

	x := ix0 + cursor_x
	if x+frame_width > sx1 {
		x = ix1 - frame_width
	}
	if x < 0 {
		x = 0
	}

	frame_height := len(*options) + 2
	if frame_height > sy1-sy0 {
		frame_height = sy1 - sy0
	}

	if t.AutoComplete, err = t.Gui.SetView("terminal.autocomplete", x, sy1-frame_height, x+frame_width, sy1-1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			panic(err)
		}
		t.AutoComplete.Frame = true
		t.AutoComplete.FrameColor = t.Input.FgColor
		t.AutoComplete.SubTitleBgColor = CurrentTheme.HelpBgColor
		t.AutoComplete.SubTitleFgColor = CurrentTheme.HelpFgColor
		t.AutoComplete.Editable = false
		t.AutoComplete.Highlight = true
		t.AutoComplete.Title = title
		t.AutoComplete.Subtitle = "\ueaa1\uea9aTAB"

		for _, option := range *options {
			text := option.Name
			p := strings.Index(option.Name, highlite)
			if p >= 0 {
				text = option.Name[:p] +
					F(t.EmFgColor) +
					option.Name[p:p+len(highlite)] +
					F(t.Screen.FgColor) +
					option.Name[p+len(highlite):]
			}
			fmt.Fprintln(t.AutoComplete, text)
		}

		t.AutoComplete.SetCursor(0, len(*options)-1)
		if len(*options) > frame_height-2 {
			t.AutoComplete.SetOrigin(0, len(*options)-frame_height+2)
		}
	}
}

func terminalEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	switch key {
	case gocui.KeyEnter:

		Terminal.HideAutocomplete()

		i := strings.TrimSpace(v.Buffer())

		fmt.Fprintln(Terminal.Screen, Terminal.FormattedPrefix+i)
		if len(Terminal.History) > 100 {
			Terminal.History = Terminal.History[1:]
		}

		if len(i) > 0 {
			if len(Terminal.History) == 0 ||
				(len(Terminal.History) > 0 &&
					Terminal.History[len(Terminal.History)-1] != i) {
				Terminal.History = append(Terminal.History, i)
			}
			Terminal.ProcessCommandFunc(v.Buffer())
		}
		Terminal.Input.Clear()

	case gocui.KeyArrowUp:
		if Terminal.AutoCompleteOn {
			ox, oy := Terminal.AutoComplete.Origin()
			cx, cy := Terminal.AutoComplete.Cursor()
			if cy > 0 {
				Terminal.AutoComplete.SetCursor(cx, cy-1)
				if cy-1 < oy {
					Terminal.AutoComplete.SetOrigin(ox, oy-1)
				}
			}
		} else {
			if strings.TrimSpace(Terminal.Input.Buffer()) == "" {
				if len(Terminal.History) > 0 {
					showHistory()
				}
			}
		}

	case gocui.KeyArrowDown:
		if Terminal.AutoCompleteOn {
			ox, oy := Terminal.AutoComplete.Origin()
			cx, cy := Terminal.AutoComplete.Cursor()
			_, ay0, _, ay1 := Terminal.AutoComplete.Dimensions()
			if cy < len(*Terminal.ACOptions)-1 {
				Terminal.AutoComplete.SetCursor(cx, cy+1)
				if cy+1-oy > ay1-ay0-3 {
					Terminal.AutoComplete.SetOrigin(ox, cy-(ay1-ay0-3))
				}
			}
		}
	case gocui.KeyTab:
		if Terminal.AutoCompleteOn {
			_, cy := Terminal.AutoComplete.Cursor()
			Terminal.Input.Clear()
			result := (*Terminal.ACOptions)[cy].Result
			fmt.Fprint(Terminal.Input, result)
			Terminal.Input.SetCursor(len(result), 0)

			// try autocomplete again
			if Terminal.AutoCompleteFunc != nil {
				t, o, h := Terminal.AutoCompleteFunc(result)
				Terminal.ShowAutocomplete(t, o, h)

			}
		}
	default:
		gocui.DefaultEditor.Edit(v, key, ch, mod)
		if Terminal.AutoCompleteFunc != nil {
			t, o, h := Terminal.AutoCompleteFunc(v.Buffer())
			Terminal.ShowAutocomplete(t, o, h)

		}
	}
}

func showHistory() {
	options := []ACOption{}
	for _, h := range Terminal.History {
		options = append(options, ACOption{Name: h, Result: h})
	}

	Terminal.ShowAutocomplete("History", &options, "")
	Terminal.AutoComplete.SetCursor(0, len(options)-1)
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
