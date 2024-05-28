package main_page

import (
	"gioui.org/example/component/icon"
	"gioui.org/layout"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"github.com/AlexNa-Holdings/web3pro/ui"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

// Page holds the state for a page demonstrating the features of
// the AppBar component.
type Page struct {
	Command component.TextField // shell command input
	*ui.Router
}

// New constructs a Page with the provided router.
func New(router *ui.Router) *Page {
	return &Page{
		Router: router,
	}
}

func (p *Page) Actions() []component.AppBarAction {
	return []component.AppBarAction{}
}

func (p *Page) Overflow() []component.OverflowAction {
	return []component.OverflowAction{}
}

func (p *Page) NavItem() component.NavItem {
	return component.NavItem{
		Name: "Pro Screen",
		Icon: icon.HomeIcon,
	}
}

func (p *Page) Layout(gtx C, th *material.Theme) D {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		p.Command_Layout(gtx),
	)
}
