package ui

import "github.com/AlexNa-Holdings/web3pro/gocui"

type Theme struct {
	Name string

	BgColor, FgColor, FrameColor          gocui.Attribute
	SelBgColor, SelFgColor, SelFrameColor gocui.Attribute

	ActionBgColor, ActionFgColor, ActionSelBgColor, ActionSelFgColor gocui.Attribute

	ErrorFgColor gocui.Attribute
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
}

var Themes = map[string]Theme{
	DarkTheme.Name:  DarkTheme,
	LightTheme.Name: LightTheme,
}

func SetTheme(g *gocui.Gui, theme string) {
	t, ok := Themes[theme]
	if !ok {
		t = DarkTheme
	}
	g.BgColor = t.BgColor
	g.FgColor = t.FgColor
	g.FrameColor = t.FrameColor
	g.SelBgColor = t.SelBgColor
	g.SelFgColor = t.SelFgColor
	g.SelFrameColor = t.SelFrameColor

	g.ActionBgColor = t.ActionBgColor
	g.ActionFgColor = t.ActionFgColor
	g.ActionSelBgColor = t.ActionSelBgColor
	g.ActionSelFgColor = t.ActionSelFgColor

	g.ErrorFgColor = t.ErrorFgColor

	CurrentTheme = &t
}
