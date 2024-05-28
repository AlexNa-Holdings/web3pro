package main_page

import (
	"unicode"

	"gioui.org/layout"
	"gioui.org/text"
	"github.com/AlexNa-Holdings/web3pro/ui"
)

func (p *Page) Command_Layout(gtx C) layout.FlexChild {
	return layout.Rigid(func(gtx C) D {

		return ui.DefaultInset.Layout(
			gtx,
			func(gtx C) D {
				if err := func() string {
					for _, r := range p.Command.Text() {
						if !unicode.IsDigit(r) {
							return "Must contain only digits"
						}
					}
					return ""
				}(); err != "" {
					p.Command.SetError(err)
				} else {
					p.Command.ClearError()
				}
				p.Command.SingleLine = true
				p.Command.Alignment = text.Start
				return p.Command.Layout(gtx, ui.UI.Theme.BasicTheme, "Command")
			},
		)
	})
}
