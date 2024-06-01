package ui

import "github.com/AlexNa-Holdings/web3pro/gocui"

type Theme struct {
	BgColor, FgColor, FrameColor          gocui.Attribute
	SelBgColor, SelFgColor, SelFrameColor gocui.Attribute

	ActionBgColor, ActionFgColor, ActionSelBgColor, ActionSelFgColor gocui.Attribute
}

var DarkTheme = Theme{
	BgColor:       gocui.GetColor("#090300"),
	FgColor:       gocui.GetColor("#A5A2A2"),
	FrameColor:    gocui.ColorWhite,
	SelBgColor:    gocui.GetColor("#090300"),
	SelFgColor:    gocui.ColorBlack,
	SelFrameColor: gocui.ColorCyan,

	ActionBgColor:    gocui.GetColor("#ffff00"),
	ActionFgColor:    gocui.GetColor("#0000cc"),
	ActionSelBgColor: gocui.ColorCyan,
	ActionSelFgColor: gocui.ColorBlack,
}

var LightTheme = Theme{
	BgColor:       gocui.ColorWhite,
	FgColor:       gocui.ColorBlack,
	FrameColor:    gocui.ColorBlack,
	SelBgColor:    gocui.ColorCyan,
	SelFgColor:    gocui.ColorBlack,
	SelFrameColor: gocui.ColorCyan,
}

var Themes = map[string]Theme{
	"dark":  DarkTheme,
	"light": LightTheme,
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
}
