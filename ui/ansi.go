package ui

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/gocui"
)

// set foreground
func F(c gocui.Attribute) string {

	r, g, b := c.RGB()

	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

// set background
func B(c gocui.Attribute) string {

	r, g, b := c.RGB()

	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

// set foreground and background
func FB(fg, bg gocui.Attribute) string {
	return F(fg) + B(bg)
}
