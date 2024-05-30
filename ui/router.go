package ui

import (
	"log"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
)

type Page interface {
	Actions() []component.AppBarAction
	Overflow() []component.OverflowAction
	Layout(gtx layout.Context, th *material.Theme) layout.Dimensions
	NavItem() component.NavItem
}

type Router struct {
	window    *app.Window
	deco      *widget.Decorations
	actions   system.Action
	decoStyle material.DecorationsStyle
	pages     map[interface{}]Page
	current   interface{}
	*component.ModalNavDrawer
	NavAnim component.VisibilityAnimation
	*component.AppBar
	*component.ModalLayer
	NonModalDrawer bool
}

func NewRouter(w *app.Window) Router {
	modal := component.NewModal()

	nav := component.NewNav("Web3 Pro", "v.0.0.0")
	modalNav := component.ModalNavFrom(&nav, modal)

	bar := component.NewAppBar(modal)
	bar.NavigationIcon = MenuIcon

	na := component.VisibilityAnimation{
		State:    component.Invisible,
		Duration: time.Millisecond * 250,
	}

	allActions := system.ActionMinimize | system.ActionMaximize | system.ActionUnmaximize |
		system.ActionClose | system.ActionMove

	deco := new(widget.Decorations)
	return Router{
		window:         w,
		pages:          make(map[interface{}]Page),
		ModalLayer:     modal,
		ModalNavDrawer: modalNav,
		AppBar:         bar,
		NavAnim:        na,
		deco:           deco,
		actions:        allActions,
		decoStyle:      material.Decorations(UI.Theme.BasicTheme, deco, allActions, "Web3 Pro"),
	}
}

func (r *Router) Register(tag interface{}, p Page) {
	r.pages[tag] = p
	navItem := p.NavItem()
	navItem.Tag = tag
	if r.current == interface{}(nil) {
		r.current = tag
		r.AppBar.Title = navItem.Name
		r.AppBar.SetActions(p.Actions(), p.Overflow())
	}
	r.ModalNavDrawer.AddNavItem(navItem)
}

func (r *Router) SwitchTo(tag interface{}) {
	p, ok := r.pages[tag]
	if !ok {
		return
	}
	navItem := p.NavItem()
	r.current = tag
	r.AppBar.Title = navItem.Name
	r.AppBar.SetActions(p.Actions(), p.Overflow())
}

func (r *Router) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {

	th = UI.Theme.BasicTheme

	for _, event := range r.AppBar.Events(gtx) {
		switch event := event.(type) {
		case component.AppBarNavigationClicked:
			if r.NonModalDrawer {
				r.NavAnim.ToggleVisibility(gtx.Now)
			} else {
				r.ModalNavDrawer.Appear(gtx.Now)
				r.NavAnim.Disappear(gtx.Now)
			}
		case component.AppBarContextMenuDismissed:
			log.Printf("Context menu dismissed: %v", event)
		case component.AppBarOverflowActionClicked:
			log.Printf("Overflow action selected: %v", event)
		}
	}
	if r.ModalNavDrawer.NavDestinationChanged() {
		r.SwitchTo(r.ModalNavDrawer.CurrentNavDestination())
	}

	// // Update the decorations based on the current window mode.
	// var actions system.Action
	// switch m := r.window.decorations.Config.Mode; m {
	// case Windowed:
	// 	actions |= system.ActionUnmaximize
	// case Minimized:
	// 	actions |= system.ActionMinimize
	// case Maximized:
	// 	actions |= system.ActionMaximize
	// case Fullscreen:
	// 	actions |= system.ActionFullscreen
	// default:
	// 	panic(fmt.Errorf("unknown WindowMode %v", m))
	// }
	r.deco.Perform(r.actions)
	// Update the window based on the actions on the decorations.
	opts, acts := splitActions(r.deco.Update(gtx))
	r.window.Option(opts...)
	r.window.Perform(acts)

	paint.Fill(gtx.Ops, th.Palette.Bg)
	content := layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Max.X /= 3
				return r.NavDrawer.Layout(gtx, th, &r.NavAnim)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return r.pages[r.current].Layout(gtx, th)
			}),
		)
	})
	bar := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return r.AppBar.Layout(gtx, th, "Menu", "Actions")
	})

	r.decoStyle.Background = th.Palette.Bg
	r.decoStyle.Foreground = th.Palette.Fg
	r.decoStyle.Title.Color = th.Palette.Fg

	flex := layout.Flex{Axis: layout.Vertical}
	flex.Layout(gtx,
		layout.Rigid(r.decoStyle.Layout), bar, content)
	r.ModalLayer.Layout(gtx, th)
	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func splitActions(actions system.Action) ([]app.Option, system.Action) {
	var opts []app.Option
	walkActions(actions, func(action system.Action) {
		switch action {
		case system.ActionMinimize:
			opts = append(opts, app.Minimized.Option())
		case system.ActionMaximize:
			opts = append(opts, app.Maximized.Option())
		case system.ActionUnmaximize:
			opts = append(opts, app.Windowed.Option())
		case system.ActionFullscreen:
			opts = append(opts, app.Fullscreen.Option())
		default:
			return
		}
		actions &^= action
	})
	return opts, actions
}

func walkActions(actions system.Action, do func(system.Action)) {
	for a := system.Action(1); actions != 0; a <<= 1 {
		if actions&a != 0 {
			actions &^= a
			do(a)
		}
	}
}
