package ui

import "github.com/AlexNa-Holdings/web3pro/gocui"

type Pane interface {
	SetView(int, int, int, int)
	GetDesc() *PaneDescriptor
	GetTemplate() string
}

type PaneDescriptor struct {
	On          bool
	MinWidth    int
	MinHeight   int
	fixed_width bool
	View        *gocui.View
}
