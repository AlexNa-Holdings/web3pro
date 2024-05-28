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

var (
	SpacingTiny   = unit.Dp(4)
	SpacingSmall  = unit.Dp(8)
	SpacingMedium = unit.Dp(16)
	SpacingLarge  = unit.Dp(32)
)

func Spacer(size unit.Dp) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Dp(size)
		gtx.Constraints.Min.Y = gtx.Dp(size)
		return layout.Dimensions{Size: gtx.Constraints.Min}
	}
}

func Init() {
	initTheams()
}
