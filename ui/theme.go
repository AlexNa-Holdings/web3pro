package ui

import (
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

type Theme struct {
	Name string

	BgColor, FgColor, FrameColor          gocui.Attribute
	SelBgColor, SelFgColor, SelFrameColor gocui.Attribute

	ActionBgColor, ActionFgColor, ActionSelBgColor, ActionSelFgColor gocui.Attribute

	ErrorFgColor gocui.Attribute
	EmFgColor    gocui.Attribute

	HelpFgColor, HelpBgColor gocui.Attribute
}

var CurrentTheme *Theme

var DarkTheme = Theme{
	Name: "dark",

	BgColor:       gocui.GetColor("#090300"),
	FgColor:       gocui.GetColor("#A5A2A2"),
	FrameColor:    gocui.GetColor("#f0f0f0"),
	SelBgColor:    gocui.GetColor("#090300"),
	SelFgColor:    gocui.GetColor("#f1f1f1"),
	SelFrameColor: gocui.ColorCyan,

	ActionBgColor:    gocui.GetColor("#ffff00"),
	ActionFgColor:    gocui.GetColor("#0000cc"),
	ActionSelBgColor: gocui.ColorCyan,
	ActionSelFgColor: gocui.ColorBlack,

	ErrorFgColor: gocui.GetColor("#ff0000"),
	EmFgColor:    gocui.ColorCyan,

	HelpFgColor: gocui.GetColor("#f1f1f1"),
	HelpBgColor: gocui.GetColor("#013220"),
}

var LightTheme = Theme{
	Name: "light",

	BgColor:       gocui.ColorWhite,
	FgColor:       gocui.ColorBlack,
	FrameColor:    gocui.ColorBlack,
	SelBgColor:    gocui.ColorCyan,
	SelFgColor:    gocui.ColorBlack,
	SelFrameColor: gocui.ColorCyan,

	ActionBgColor:    gocui.GetColor("#ffff00"),
	ActionFgColor:    gocui.GetColor("#0000cc"),
	ActionSelBgColor: gocui.ColorCyan,
	ActionSelFgColor: gocui.ColorBlack,

	ErrorFgColor: gocui.GetColor("#ff0000"),
	EmFgColor:    gocui.ColorCyan,

	HelpFgColor: gocui.GetColor("#f1f1f1"),
	HelpBgColor: gocui.GetColor("#013220"),
}

var Themes = map[string]Theme{
	DarkTheme.Name:  DarkTheme,
	LightTheme.Name: LightTheme,
}

func SetTheme(theme string) {
	t, ok := Themes[theme]
	if !ok {
		log.Error().Msgf("Unknown theme: %s", theme)
		t = DarkTheme
	}
	Gui.BgColor = t.BgColor
	Gui.FgColor = t.FgColor
	Gui.FrameColor = t.FrameColor
	Gui.SelBgColor = t.SelBgColor
	Gui.SelFgColor = t.SelFgColor
	Gui.SelFrameColor = t.SelFrameColor

	Gui.ActionBgColor = t.ActionBgColor
	Gui.ActionFgColor = t.ActionFgColor
	Gui.ActionSelBgColor = t.ActionSelBgColor
	Gui.ActionSelFgColor = t.ActionSelFgColor

	Gui.ErrorFgColor = t.ErrorFgColor
	Gui.EmFgColor = t.EmFgColor

	Gui.SubTitleFgColor = t.BgColor
	Gui.SubTitleBgColor = t.EmFgColor

	CurrentTheme = &t

}
