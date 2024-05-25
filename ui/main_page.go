package ui

import (
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type (
	D = layout.Dimensions
	C = layout.Context
)

type Terminal struct {
	input  *widget.Editor
	output string
}

var (
	term = &Terminal{
		input: &widget.Editor{
			SingleLine: true,
			Submit:     true,
		},
	}
)

func MainPage(gtx layout.Context) layout.Dimensions {
	th := UI.Theme.BasicTheme

	for {
		e, ok := term.input.Update(gtx)
		if !ok {
			break
		}
		if e, ok := e.(widget.SubmitEvent); ok {
			term.output += e.Text + "\n"
			term.input.SetText("")
		}
	}

	// Layout the terminal UI.

	paint.Fill(gtx.Ops, th.Palette.Bg)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			//border := widget.Border{Color: th.Palette.Bg, CornerRadius: unit.Dp(4), Width: unit.Dp(2)}
			//return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			in := layout.UniformInset(unit.Dp(8))
			return in.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(th, term.output)
				lbl.Alignment = text.Start
				return lbl.Layout(gtx)
			})
			//})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(2), Left: unit.Dp(2), Right: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				border := widget.Border{Color: UI.Theme.BorderColor, CornerRadius: unit.Dp(4), Width: unit.Dp(2)}
				return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					e := material.Editor(th, term.input, "Enter command")
					return layout.UniformInset(unit.Dp(8)).Layout(gtx, e.Layout)
				})
			})
		}),
	)
}
