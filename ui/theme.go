package ui

import (
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

type ThemeStruct struct {
	Name string

	BgColor, FgColor, FrameColor          gocui.Attribute
	SelBgColor, SelFgColor, SelFrameColor gocui.Attribute

	ActionBgColor, ActionFgColor, ActionSelBgColor, ActionSelFgColor gocui.Attribute

	ErrorFgColor gocui.Attribute
	EmFgColor    gocui.Attribute
	InputBgColor gocui.Attribute

	HelpFgColor, HelpBgColor gocui.Attribute
}

var Theme *ThemeStruct

var DarkTheme = ThemeStruct{
	Name: "dark",

	BgColor:       gocui.GetColor("#090300"),
	FgColor:       gocui.GetColor("#A5A2A2"),
	FrameColor:    gocui.GetColor("#a7a7a7"),
	SelBgColor:    gocui.GetColor("#090300"),
	SelFgColor:    gocui.GetColor("#f3f3f3"),
	SelFrameColor: gocui.GetColor("#7FFFD4"),

	ActionBgColor:    gocui.GetColor("#FFD700"),
	ActionFgColor:    gocui.GetColor("#0000cc"),
	ActionSelBgColor: gocui.ColorCyan,
	ActionSelFgColor: gocui.ColorBlack,

	ErrorFgColor: gocui.GetColor("#ff0000"),
	EmFgColor:    gocui.ColorCyan,

	HelpFgColor:  gocui.GetColor("#f1f1f1"),
	HelpBgColor:  gocui.GetColor("#028002"),
	InputBgColor: gocui.GetColor("#292320"),
}

var LightTheme = ThemeStruct{
	Name: "light",

	BgColor:       gocui.GetColor("#f1f1f1"),
	FgColor:       gocui.ColorBlack,
	FrameColor:    gocui.ColorBlack,
	SelBgColor:    gocui.ColorCyan,
	SelFgColor:    gocui.ColorBlack,
	SelFrameColor: gocui.ColorCyan,

	ActionBgColor:    gocui.ColorCyan,
	ActionFgColor:    gocui.GetColor("#ffff00"),
	ActionSelBgColor: gocui.ColorCyan,
	ActionSelFgColor: gocui.ColorBlack,

	ErrorFgColor: gocui.GetColor("#ff0000"),
	EmFgColor:    gocui.ColorCyan,

	HelpFgColor:  gocui.GetColor("#f1f1f1"),
	HelpBgColor:  gocui.GetColor("#013220"),
	InputBgColor: gocui.GetColor("#717171"),
}

var Themes = map[string]ThemeStruct{
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
	Gui.InputBgColor = t.InputBgColor

	Gui.HelpFgColor = t.HelpFgColor
	Gui.HelpBgColor = t.HelpBgColor

	Gui.JoinedFrameBgColor = t.BgColor
	Gui.JoinedFrameFgColor = t.FrameColor

	Theme = &t

	cmn.Config.Theme = theme
	cmn.ConfigChanged = true

	log.Info().Msgf("Theme set to: %s", theme)

}
