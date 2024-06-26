package ui

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
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
var TimerQuit = make(chan bool)

func add(hail *cmn.HailRequest) {
	Mutex.Lock()
	defer Mutex.Unlock()

	if hail.Priorized {
		HailQueue = append([]*cmn.HailRequest{hail}, HailQueue...)
	} else {
		HailQueue = append(HailQueue, hail)
	}
}

func remove(hail *cmn.HailRequest) {
	Mutex.Lock()
	defer Mutex.Unlock()

	for i, h := range HailQueue {
		if h == hail {
			if hail.OnClose != nil {
				hail.OnClose()
			}

			HailQueue = append(HailQueue[:i], HailQueue[i+1:]...)

			if len(HailQueue) > 0 {
				HailPane.open(HailQueue[0])
			} else {
				Gui.UpdateAsync(func(g *gocui.Gui) error {
					Gui.DeleteView("hail")
					HailPane.View = nil
					// stop timer
					TimerQuit <- true
					return nil
				})
			}

			return
		}
	}
}

func cancel(hail *cmn.HailRequest) {
	if hail.OnCancel != nil {
		hail.OnCancel()
	}
	remove(hail)
}

func HailPaneTimer() {
	for {
		select {
		case <-TimerQuit:
			return
		case <-time.After(1 * time.Second):
			if ActiveRequest != nil {
				if time.Until(ActiveRequest.Expiration) < 0 {
					cancel(ActiveRequest)
				} else {
					Gui.UpdateAsync(func(g *gocui.Gui) error {
						HailPane.UpdateSubtitle()
						return nil
					})
				}
			}
		}
	}
}

func ProcessHails() {
	for {
		hail := <-cmn.HailChannel

		add(hail)

		if ActiveRequest != HailQueue[0] {
			if ActiveRequest != nil {
				if ActiveRequest.OnSuspend != nil {
					ActiveRequest.OnSuspend()
				}
				ActiveRequest.Suspended = true
			}
		}

		HailPane.open(HailQueue[0])
	}
}

func (p *HailPaneType) open(hail *cmn.HailRequest) {
	ActiveRequest = hail

	if hail.Suspended {
		if hail.OnResume != nil {
			hail.OnResume()
		}
		hail.Suspended = false
	}

	if HailPane.View == nil {
		HailPane.SetView(Gui, 0, 0, 10, 10)
		go HailPaneTimer()
	}

	hail.Expiration = time.Now().Add(time.Duration(hail.TimeoutSec) * time.Second)

	Gui.UpdateAsync(func(g *gocui.Gui) error {
		HailPane.UpdateSubtitle()
		return nil
	})

	if hail.OnOpen != nil {
		hail.OnOpen(Gui, HailPane.View)
	}
}

func (p *HailPaneType) UpdateSubtitle() {
	if HailPane.View != nil && ActiveRequest != nil {

		left := time.Until(ActiveRequest.Expiration)

		if left.Seconds() < 10 {
			p.SubTitleBgColor = Theme.ErrorFgColor
		} else {
			p.SubTitleBgColor = Theme.HelpBgColor
		}

		left = left.Round(time.Second)

		if len(HailQueue) > 1 {
			p.Subtitle = fmt.Sprintf("(%d) %s", len(HailQueue), left.String())
		} else {
			p.Subtitle = left.String()
		}

		HailPane.View.Subtitle = p.Subtitle
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
		if ActiveRequest.Title != "" {
			p.View.Title = ActiveRequest.Title
		}
		p.View.SubTitleFgColor = Theme.HelpFgColor
		p.View.SubTitleBgColor = Theme.HelpBgColor
		p.View.FrameColor = Gui.ActionBgColor
		p.View.TitleColor = Gui.ActionFgColor
		p.View.EmFgColor = Gui.ActionBgColor
		p.View.ScrollBar = true
		p.View.OnResize = func(v *gocui.View) {
			x0, y0, x1, y1 := v.Dimensions()
			log.Debug().Msgf("HailPane resized to %d %d %d %d", x0, y0, x1, y1)
			if ActiveRequest != nil {
				if ActiveRequest.Template != "" {
					v.RenderTemplate(ActiveRequest.Template)
				}
			}
		}
		p.View.OnClickTitle = func(v *gocui.View) { // reset timer
			if ActiveRequest != nil {
				ActiveRequest.Expiration = ActiveRequest.Expiration.Add(time.Duration(ActiveRequest.TimeoutSec) * time.Second)
			}

			Gui.UpdateAsync(func(g *gocui.Gui) error {
				HailPane.UpdateSubtitle()
				return nil
			})
		}
	}
}
