package settings

import (
	"gioui.org/example/component/icon"
	"gioui.org/layout"
	"gioui.org/widget"
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
	widget.List
	*ui.Router
	nonModalDrawer widget.Bool
	themeEnum      widget.Enum
}

// New constructs a Page with the provided router.
func New(router *ui.Router) *Page {
	return &Page{
		Router:    router,
		themeEnum: widget.Enum{Value: ui.GetThemeName()},
	}
}

// var _ ui.Page = &Page{}

func (p *Page) Actions() []component.AppBarAction {
	return []component.AppBarAction{}
}

func (p *Page) Overflow() []component.OverflowAction {
	return []component.OverflowAction{}
}

func (p *Page) NavItem() component.NavItem {
	return component.NavItem{
		Name: "Settings",
		Icon: icon.SettingsIcon,
	}
}

func (p *Page) Layout(gtx C, th *material.Theme) D {
	p.List.Axis = layout.Vertical
	return material.List(th, &p.List).Layout(gtx, 1, func(gtx C, _ int) D {
		return layout.Flex{
			Alignment: layout.Middle,
			Axis:      layout.Vertical,
		}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return ui.DefaultInset.Layout(gtx, material.H5(th, `User interface`).Layout)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.Row2{}.Layout(gtx, material.Body1(th, "Use non-modal drawer").Layout,
					func(gtx C) D {
						if p.nonModalDrawer.Update(gtx) {
							p.Router.NonModalDrawer = p.nonModalDrawer.Value
							if p.nonModalDrawer.Value {
								p.Router.NavAnim.Appear(gtx.Now)
							} else {
								p.Router.NavAnim.Disappear(gtx.Now)
							}
						}
						return material.Switch(th, &p.nonModalDrawer, "Use Non-Modal Navigation Drawer").Layout(gtx)
					})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {

				if p.themeEnum.Update(gtx) {
					ui.SetTheme(p.themeEnum.Value)
					// ui.Router.
				}

				return ui.Row2{}.Layout(gtx, material.Body1(th, "Theme").Layout,
					func(gtx C) D {
						return layout.Flex{Axis: layout.Horizontal}.Layout(
							gtx,
							layout.Rigid(func(gtx C) D {
								return material.RadioButton(
									th,
									&p.themeEnum,
									"dark",
									"Dark",
								).Layout(gtx)
							}),
							layout.Rigid(func(gtx C) D {
								return material.RadioButton(
									th,
									&p.themeEnum,
									"light",
									"Light",
								).Layout(gtx)
							}),
						)
					})
			}),
		)

	})
}
