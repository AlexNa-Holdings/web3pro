package ui

import (
	"image/color"

	"gioui.org/font/gofont"
	"gioui.org/text"
	"gioui.org/widget/material"
)

type UITheme struct {
	Name        string
	BasicTheme  *material.Theme
	BorderColor color.NRGBA
}

var LightTheme, DarkTheme UITheme

func initTheams() {

	// make a monospace font

	sharper := text.NewShaper(text.WithCollection(gofont.Collection()))

	// LightTheme
	LightTheme.Name = "light"
	LightTheme.BasicTheme = material.NewTheme()
	LightTheme.BasicTheme.Palette = material.Palette{
		Fg:         color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF},
		Bg:         color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
		ContrastBg: color.NRGBA{R: 0x3f, G: 0x51, B: 0xb5, A: 0xff},
		ContrastFg: color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	}
	LightTheme.BasicTheme.Shaper = sharper
	LightTheme.BasicTheme.Face = "monospace"
	LightTheme.BorderColor = color.NRGBA{R: 0xF, G: 0x0F, B: 0x0F, A: 0xFF}

	// DarkTheme
	DarkTheme.Name = "dark"
	DarkTheme.BasicTheme = material.NewTheme()
	DarkTheme.BasicTheme.Palette = material.Palette{
		Fg:         color.NRGBA{R: 0xCF, G: 0xCF, B: 0xD4, A: 0xFF},
		Bg:         color.NRGBA{R: 0x17, G: 0x17, B: 0x1c, A: 0xFF},
		ContrastBg: color.NRGBA{R: 0x3f, G: 0x51, B: 0xb5, A: 0xFF},
		ContrastFg: color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	}

	DarkTheme.BasicTheme.Shaper = sharper
	DarkTheme.BasicTheme.Face = "monospace"
	DarkTheme.BorderColor = color.NRGBA{R: 0xFF, G: 0xF0, B: 0xF0, A: 0xF0}

	UI.Theme = &LightTheme
}

func GetThemeName() string {
	return UI.Theme.Name
}

func SetTheme(name string) {
	switch name {
	case "light":
		UI.Theme = &LightTheme
	case "dark":
		UI.Theme = &DarkTheme
	default:
		UI.Theme = &DarkTheme
	}
}
