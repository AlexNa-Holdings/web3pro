package gocui

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type PUCType int // PopupControlType

type Popup struct {
	*View
	Name            string
	Width, Height   int
	Title, Subtitle string
	Template        string
	GlgLayout       func(g *Gui) error
	OnOverHotspot   func(v *View, hs *Hotspot)
	OnClickHotspot  func(v *View, hs *Hotspot)
	OnClose         func(v *View)
	OnOpen          func(v *View)
}

func (g *Gui) ShowPopup(p *Popup) {
	g.popup = p
}

func (g *Gui) HidePopup() {
	if g.popup != nil {

		if g.popup.OnClose != nil {
			g.popup.OnClose(g.popup.View)
		}

		for _, c := range g.popup.View.Controls {
			if c.Type == PUC_INPUT {
				g.DeleteView(c.View.name)
			}
		}

		g.DeleteView(g.popup.Name)
		g.popup = nil
	}
}

func (p *Popup) Layout(g *Gui) error {
	maxX, maxY := g.Size()

	if p.Name == "" {
		p.Name = "popup"
	}

	// Center the popup
	if p.Height == 0 {
		p.Height = len(strings.Split(p.Template, "\n")) + 2
	}

	if p.Width == 0 {
		// calc the longest line
		lines := strings.Split(p.Template, "\n")
		for _, line := range lines {
			l := p.calcLineWidth(line)
			if l > p.Width {
				p.Width = l
			}
		}

		p.Width += 2
	}

	if v, err := g.SetView(p.Name, maxX/2-p.Width/2, maxY/2-p.Height/2, maxX/2+p.Width/2, maxY/2+p.Height/2, 0); err != nil {
		if !errors.Is(err, ErrUnknownView) {
			return err
		}

		v.FrameRunes = []rune{'═', '║', '╔', '╗', '╚', '╝'}
		v.Frame = true
		v.Title = p.Title
		v.Subtitle = p.Subtitle
		v.Editable = false
		v.OnOverHotspot = func(v *View, hs *Hotspot) {
			if g.popup.OnOverHotspot != nil {
				g.popup.OnOverHotspot(v, hs)
			}
		}
		v.OnClickHotspot = func(v *View, hs *Hotspot) {
			if g.popup.OnClickHotspot != nil {
				g.popup.OnClickHotspot(v, hs)
			}
		}

		g.popup.View = v
		g.SetCurrentView(v.name)
		switch {
		case p.Template != "":
			err = p.ParseTemplate()
		case p.GlgLayout != nil:
			err = p.GlgLayout(g)
		default:
			err = errors.New("no template or layout function")
		}
		if err != nil {
			return err
		}

		if p.OnOpen != nil {
			p.OnOpen(v)
		}

		v.SetFocus(0)

	}

	return nil
}

func (p *Popup) ParseTemplate() error {

	re := regexp.MustCompile(`<(\w+)((?:\s+\w+(?::(?:\w+|"[^"]*"))?\s*)*)>`)
	lines := strings.Split(p.Template, "\n")

	if len(lines) == 0 {
		return errors.New("empty template")
	}

	for _, line := range lines {
		matches := re.FindAllStringIndex(line, -1)

		left := 0

		for _, match := range matches {
			fmt.Fprint(p.View, line[left:match[0]])
			tag := line[match[0]:match[1]]
			if tag == "<c>" { // center
				if left != 0 {
					return errors.New("center tag must be at the beginning of the line")
				}

				n := (p.Width - p.calcLineWidth(line) - 2) / 2
				for i := 0; i < n; i++ {
					fmt.Fprint(p.View, " ")
				}
			} else {
				p.View.AddTag(tag)
			}
			left = match[1]
		}

		fmt.Fprintln(p.View, line[left:])

	}

	return nil
}

func (p *Popup) calcLineWidth(line string) int {
	l := len(line)
	re := regexp.MustCompile(`<(\w+)((?:\s+\w+(?::(?:\w+|"[^"]*"))?\s*)*)>`)
	matches := re.FindAllStringIndex(line, -1)
	for _, match := range matches {
		tag := line[match[0]:match[1]]
		tagName, tagParams := ParseTag(tag)
		l = l - len(tag) + GetTagLength(tagName, tagParams)
	}
	return l
}

func (p *Popup) GetInput(id string) string {
	f, err := p.View.gui.View(p.View.name + "." + id)
	if err != nil {
		return ""
	}

	return f.Buffer()
}

func (p *Popup) SetInput(id, value string) {
	f, err := p.View.gui.View(p.View.name + "." + id)
	if err != nil {
		return
	}

	f.Clear()
	fmt.Fprint(f, value)
}
