package ui

import "github.com/AlexNa-Holdings/web3pro/gocui"

type Theme struct {
	BgColor, FgColor, FrameColor          gocui.Attribute
	SelBgColor, SelFgColor, SelFrameColor gocui.Attribute
}

var DarkTheme = Theme{
	BgColor:       gocui.ColorBlack,
	FgColor:       gocui.ColorWhite,
	FrameColor:    gocui.ColorWhite,
	SelBgColor:    gocui.ColorCyan,
	SelFgColor:    gocui.ColorBlack,
	SelFrameColor: gocui.ColorCyan,
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
}
