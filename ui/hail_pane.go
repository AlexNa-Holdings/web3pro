package ui

import (
	"errors"
	"sync"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
)

type HailPaneType struct {
	*gocui.View
	MinWidth  int
	MinHeight int
}

var HailPane *HailPaneType = &HailPaneType{
	MinWidth:  30,
	MinHeight: 9,
}

var ActiveRequest *cmn.HailRequest
var HailQueue []*cmn.HailRequest
var Mutex = &sync.Mutex{}

func add(request *cmn.HailRequest) {
	Mutex.Lock()
	defer Mutex.Unlock()

	if request.Priorized {
		HailQueue = append([]*cmn.HailRequest{request}, HailQueue...)
	} else {
		HailQueue = append(HailQueue, request)
	}
}

func ProcessHails() {
	for {
		request := <-cmn.HailChannel

		add(request)

		if ActiveRequest != HailQueue[0] {
			if ActiveRequest != nil {
				if ActiveRequest.OnSuspend != nil {
					ActiveRequest.OnSuspend()
				}
				ActiveRequest.Suspended = true
			}
		}

		open(HailQueue[0])
	}

}

func open(request *cmn.HailRequest) {
	ActiveRequest = request

	if request.Suspended {
		if request.OnResume != nil {
			request.OnResume()
		}
		request.Suspended = false
	}

	if HailPane.View == nil {
		HailPane.SetView(Gui, 0, 0, 1, 1)
		Gui.Update(func(g *gocui.Gui) error {
			return nil
		})
	}

	if request.OnOpen != nil {
		request.OnOpen(Gui, HailPane.View)
	}
}

func (p *HailPaneType) SetView(g *gocui.Gui, x0, y0, x1, y1 int) {
	var err error

	if ActiveRequest == nil {
		return
	}

	if p.View, err = g.SetView("hail", x0, y0, x1, y1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			panic(err)
		}
		p.View.Title = "Hail"
		p.View.Autoscroll = true
		p.View.SubTitleFgColor = CurrentTheme.BgColor
		p.View.SubTitleBgColor = CurrentTheme.FrameColor
		p.View.FrameColor = Gui.ActionBgColor
		p.View.TitleColor = Gui.ActionFgColor
		p.View.EmFgColor = Gui.ActionBgColor
		p.View.OnResize = func(v *gocui.View) {
			if ActiveRequest != nil {
				if ActiveRequest.Template != "" {
					v.RenderTemplate(ActiveRequest.Template)
				}
			}
		}
		p.Subtitle = "(1)"
	}
}
