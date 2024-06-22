package gocui

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
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
	ComboList       *View
}

func (g *Gui) ShowPopup(p *Popup) {
	g.popup = p
}

func (g *Gui) HidePopup() {
	if g.popup != nil {

		if g.popup.View.activeHotspot != nil {
			g.popup.View.activeHotspot = nil
			if g.popup.View.OnOverHotspot != nil {
				g.popup.View.OnOverHotspot(g.popup.View, nil)
			}
		}

		if g.popup.OnClose != nil {
			g.popup.OnClose(g.popup.View)
		}

		for _, c := range g.popup.View.Controls {
			if c.Type == PUC_INPUT || c.Type == PUC_TEXT_INPUT {
				g.DeleteView(c.View.name)
			}
		}

		g.DeleteView(g.popup.Name)
		g.popup = nil
	}
}

func (p *Popup) Layout(g *Gui) error {
	var v *View
	var err error

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

		// p.Width += 2
	}

	if v, err = g.SetView(p.Name, maxX/2-p.Width/2, maxY/2-p.Height/2, maxX/2+p.Width/2, maxY/2+p.Height/2, 0); err != nil {
		if !errors.Is(err, ErrUnknownView) {
			return err
		}

		v.FrameRunes = []rune{'═', '║', '╔', '╗', '╚', '╝'}
		v.FrameColor = g.EmFgColor
		v.TitleColor = g.EmFgColor
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
			// Hande comboboxes
			if hs != nil {
				params := strings.Split(hs.Value, " ")

				if len(params) >= 2 && params[0] == "combobox" {
					for _, c := range v.Controls {
						if c.Type == PUC_COMBOBOX && c.name == v.name+"."+params[1] {

							if p.ComboList != nil {
								g.DeleteView(p.ComboList.name)
								p.ComboList = nil
							} else {
								width := 8 // minimum width

								log.Debug().Msgf("????? %s %v", c.name, c.Items)

								for _, item := range c.Items {
									if len(item) > width {
										width = len(item) + 2
									}
								}

								if width > c.x1-c.x0 {
									width = c.x1 - c.x0
								}

								height := len(c.Items)
								if height > v.y1-v.y0-2 {
									height = v.y1 - v.y0 - 2
								}

								if height == 0 {
									height = 1
								}

								height += 1

								x0 := v.x0 + c.x1 - width
								if x0 < v.x0 {
									x0 = v.x0
								}

								x1 := v.x0 + c.x1

								y0 := v.y0 + c.y1
								if y0+height >= v.y1 {
									y0 = v.y1 - height
								}

								y1 := y0 + height
								if y1 >= v.y1 {
									y1 = v.y1
								}

								if p.ComboList, err = g.SetView(c.View.name+".list", x0, y0, x1, y1, 0); err != nil {
									p.ComboList.Frame = true
									p.ComboList.Editable = false
									p.ComboList.Wrap = false
									p.ComboList.Highlight = true
									// p.ComboList.SelBgColor = g.EmBgColor
									// p.ComboList.SelFgColor = g.EmFgColor
									for _, item := range c.Items {
										fmt.Fprintln(p.ComboList, item)
									}
									g.SetViewOnTop(p.ComboList.name)
									g.SetCurrentView(p.ComboList.name)
								}
							}
						}
					}
				}
			}

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

		v.SetFocus(0)

		if p.OnOpen != nil {
			p.OnOpen(v)
		}

	}

	for _, c := range p.Controls {
		if c.Type == PUC_INPUT {
			c.View.x0 = v.x0 + c.x0
			c.View.x1 = v.x0 + c.x1
			c.View.y0 = v.y0 + c.y0
			c.View.y1 = v.y0 + c.y1
			c.View.tainted = true
			g.SetViewOnTop(c.View.name)
		}
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

				n := (p.Width-2-p.calcLineWidth(line))/2 + 1
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
