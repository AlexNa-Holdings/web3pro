package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

type TerminalPane struct {
	PaneDescriptor
	Screen           *gocui.View
	Input            *gocui.View
	Prefix           *gocui.View
	AutoComplete     *gocui.View
	AutoCompleteOn   bool
	ACOptions        *[]ACOption // autocomplete options
	ACTitle          string      //autocomplete title
	ACHighlite       string      //autocomplete highlite
	CommandPrefix    string
	FormattedPrefix  string
	AutoCompleteFunc func(string) (string, *[]ACOption, string)
	History          []string
	On               bool
}

type ACOption struct {
	Name   string
	Result string
}

const DEFAULT_COMMAND_PREFIX = "w3p"

var Terminal TerminalPane = TerminalPane{
	CommandPrefix:  DEFAULT_COMMAND_PREFIX,
	History:        []string{},
	ACOptions:      &[]ACOption{},
	AutoCompleteOn: false,
	On:             true,
	PaneDescriptor: PaneDescriptor{
		MinWidth:               10000, // 10000 is a hack to make it take all the space
		MinHeight:              1,
		SupportCachedHightCalc: false,
	},
}

func (p *TerminalPane) GetDesc() *PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *TerminalPane) EstimateLines(w int) int {
	return 0
}

func (p *TerminalPane) IsOn() bool {
	return p.On
}

func (p *TerminalPane) SetOn(on bool) {
	p.On = on
}

func (p *TerminalPane) SetScreenView(x0, y0, x1, y1 int) {
	v, err := Gui.SetView("terminal.screen", x0, y0, x1, y1, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}
		v.Autoscroll = false
		v.Wrap = false
		v.Frame = false
		v.Highlight = false
		v.Editable = false
		v.ScrollBar = true
		v.OnOverHotspot = ProcessOnOverHotspot
		v.OnClickHotspot = ProcessOnClickHotspot
	}
	p.Screen = v
}

func (p *TerminalPane) SetInputView(x0, y0, x1, y1 int) {
	v, err := Gui.SetView("terminal.input", x0, y0, x1, y1, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}
		v.Frame = false
		v.Editable = true
		v.Highlight = false
		v.Editor = gocui.EditorFunc(terminalEditor)
		v.Autoscroll = true
		Gui.SetCurrentView("terminal.input")
	}
	p.Input = v
}

func (p *TerminalPane) SetPrefixView(x0, y0, x1, y1 int) {
	v, err := Gui.SetView("terminal.prefix", x0, y0, x1, y1, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}
		v.Frame = false

		p.FormattedPrefix = p.formatPrefix(p.Screen.FgColor, p.Screen.BgColor)

		fmt.Fprint(v, p.formatPrefix(v.FgColor, v.BgColor))
		Gui.SetViewOnTop("terminal.prefix")
	}
	p.Prefix = v
}

func (p *TerminalPane) SetView(x0, y0, x1, y1 int, overlap byte) {
	v, err := Gui.SetView("terminal", x0, y0, x1, y1, overlap)
	if err != nil {
		p.PaneDescriptor.View = v
		v.JoinedFrame = true
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}
		v.Title = "Terminal"
	}

	p.SetScreenView(x0, y0, x1, y1-1)
	prefix_len := len(p.CommandPrefix) + 1

	p.SetInputView(x0+prefix_len, y1-2, x1, y1)
	p.SetPrefixView(x0, y1-2, x0+prefix_len+1, y1)

	if p.AutoCompleteOn {
		p.layoutAutocomplete(p.ACTitle, p.ACOptions, p.ACHighlite)
	}
}

func (t *TerminalPane) SetCommandPrefix(prefix string) {
	t.CommandPrefix = prefix
	t.FormattedPrefix = t.formatPrefix(t.Screen.FgColor, t.Screen.BgColor)
	t.Prefix.Clear()
	fmt.Fprint(t.Prefix, t.FormattedPrefix)
}

func (t *TerminalPane) formatPrefix(fgColor, bgColor gocui.Attribute) string {
	return FB(Gui.ActionFgColor, Gui.ActionBgColor) +
		t.CommandPrefix +
		FB(Gui.ActionBgColor, bgColor) +
		"\ue0b0" +
		FB(fgColor, bgColor)
}

func (t *TerminalPane) ShowAutocomplete(title string, options *[]ACOption, highlite string) {
	t.HideAutocomplete()

	if options != nil && len(*options) > 0 {

		t.ACOptions = options
		t.ACTitle = title
		t.ACHighlite = highlite
		t.AutoCompleteOn = true
	}
}

func (p *TerminalPane) HideAutocomplete() {
	if p.AutoCompleteOn {
		Gui.DeleteView("terminal.autocomplete")
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
	input := t.Input.Buffer()
	ix0, _, ix1, _ := t.Input.Dimensions()
	io0, _ := t.Input.Origin()
	_ /*sy0*/, _, sx1, sy1 := t.Screen.Dimensions()

	frame_width := max(longest_option+2, len(title)+16) // make the title visible

	x := ix0 + len(input) - io0
	if x+frame_width > sx1 {
		x = ix1 - frame_width
	}
	if x < 0 {
		x = 0
	}

	frame_height := len(*options) + 2
	// if frame_height > sy1-sy0 {
	// 	frame_height = sy1 - sy0
	// }
	if frame_height > sy1 {
		frame_height = sy1
	}

	if t.AutoComplete, err = Gui.SetView("terminal.autocomplete", x, sy1-frame_height, x+frame_width, sy1-1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}
		t.AutoComplete.Frame = true
		t.AutoComplete.FrameColor = t.Input.FgColor
		t.AutoComplete.SubTitleBgColor = Theme.HelpBgColor
		t.AutoComplete.SubTitleFgColor = Theme.HelpFgColor
		t.AutoComplete.Editable = false
		t.AutoComplete.Highlight = true
		t.AutoComplete.Title = title
		t.AutoComplete.ScrollBar = true
		if len(*options) > 1 {
			t.AutoComplete.Subtitle = "\uf431\uf433\uf432"
		} else {
			t.AutoComplete.Subtitle = "\uf432"
		}

		for _, option := range *options {
			text := option.Name

			p := strings.Index(strings.ToLower(option.Name), strings.ToLower(highlite))
			if p >= 0 {
				text = option.Name[:p] +
					F(Gui.EmFgColor) +
					option.Name[p:p+len(highlite)] +
					F(Gui.FgColor) +
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
			bus.Send("ui", "command", v.Buffer())
		}
		Terminal.Screen.ScrollBottom()
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
	case gocui.KeyTab, gocui.KeyArrowRight:
		if Terminal.AutoCompleteOn {
			_, cy := Terminal.AutoComplete.Cursor()

			if cy >= 0 && cy < len(*Terminal.ACOptions) {
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
		} else {
			gocui.DefaultEditor.Edit(v, key, ch, mod)
		}
	default:
		gocui.DefaultEditor.Edit(v, key, ch, mod)
		if Terminal.AutoCompleteFunc != nil {
			t, o, h := Terminal.AutoCompleteFunc(v.Buffer())
			Terminal.ShowAutocomplete(t, o, h)

		}
	}
}

func OnAutocompleteMouseDown(g *gocui.Gui, v *gocui.View) error {
	_, cy := Terminal.AutoComplete.Cursor()

	if cy >= 0 && cy < len(*Terminal.ACOptions) {
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

	return nil
}

func showHistory() {
	options := []ACOption{}
	for _, h := range Terminal.History {
		options = append(options, ACOption{Name: h, Result: h})
	}
	Terminal.ShowAutocomplete("History", &options, "")
}

func PrintErrorf(format string, a ...interface{}) {
	str := fmt.Sprintf(format, a...)
	Terminal.Screen.Write([]byte("\n" + F(Theme.ErrorFgColor) + str + F(Terminal.Screen.FgColor) + "\n\n"))
}

func Printf(format string, a ...interface{}) {
	str := fmt.Sprintf(format, a...)
	Terminal.Screen.Write([]byte(str))
}

func ResetColors() {
	Terminal.Screen.Write([]byte(FB(Terminal.Screen.FgColor, Terminal.Screen.BgColor)))
}

func Flush() {
	Gui.Update(func(g *gocui.Gui) error {
		Terminal.Screen.ScrollBottom()
		return nil
	})
}
