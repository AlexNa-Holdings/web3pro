package gocui

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type Popup struct {
	*View
	Width, Height   int
	Title, Subtitle string
	Template        string
	GlgLayout       func(g *Gui) error
}

func (g *Gui) ShowPopup(p *Popup) {
	g.popup = p
}

func (p *Popup) Layout(g *Gui) error {
	maxX, maxY := g.Size()

	// Center the popup

	if v, err := g.SetView("popup", maxX/2-p.Width/2, maxY/2-p.Height/2, maxX/2+p.Width/2, maxY/2+p.Height/2, 0); err != nil {
		if !errors.Is(err, ErrUnknownView) {
			return err
		}

		v.FrameRunes = []rune{'═', '║', '╔', '╗', '╚', '╝'}
		v.Frame = true
		v.Title = p.Title
		v.Subtitle = p.Subtitle
		g.popup.View = v
		g.SetCurrentView("popup")
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
			p.View.AddTag(tag)
			left = match[1]
		}

		fmt.Fprintln(p.View, line[left:])

	}

	return nil
}
