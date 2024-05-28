package ui

import (
	"gioui.org/layout"
	"gioui.org/unit"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

var DefaultInset = layout.UniformInset(unit.Dp(8))

var UI struct {
	Theme *UITheme
}

func Init() {
	initTheams()
}
