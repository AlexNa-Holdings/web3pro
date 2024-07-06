package ui

import (
	"errors"
	"fmt"
	"strings"
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

func add(hail *cmn.HailRequest) bool { // returns if on top
	log.Trace().Msgf("Adding hail: %s", hail.Title)
	Mutex.Lock()
	defer Mutex.Unlock()

	if hail.Priorized {
		HailQueue = append([]*cmn.HailRequest{hail}, HailQueue...)
	} else {
		HailQueue = append(HailQueue, hail)
	}

	log.Debug().Msgf("Hail added signal sent: %s on top: %b", hail.Title, hail == HailQueue[0])

	return hail == HailQueue[0]
}

func remove(hail *cmn.HailRequest) {
	log.Trace().Msgf("Removing hail: %s", hail.Title)

	Mutex.Lock()
	for i, h := range HailQueue {
		if h == hail {
			HailQueue = append(HailQueue[:i], HailQueue[i+1:]...)
		}
	}
	Mutex.Unlock()

	log.Debug().Msgf("Hail removed: %s  n_left: %d", hail.Title, len(HailQueue))

	if hail.OnClose != nil {
		hail.OnClose(hail)
	}

	log.Debug().Msgf("Hail removed signal sent: %s", hail.Title)
	hail.Done <- true
	log.Debug().Msgf("After hail removed signal sent: %s", hail.Title)

	if ActiveRequest == hail { // we closed the active request
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
	}
}

func cancel(hail *cmn.HailRequest) {
	log.Debug().Msgf("Cancel: hail: %s", hail.Title)
	if hail.OnCancel != nil {
		hail.OnCancel(hail)
	}
	remove(hail)
}

func HailPaneTimer() {
	tick := 0

	for {
		select {
		case <-TimerQuit:
			return
		case <-time.After(1 * time.Second):
			tick++
			if ActiveRequest != nil && !ActiveRequest.TimerPaused {
				if time.Until(ActiveRequest.Expiration) <= 0 {
					cancel(ActiveRequest)
				} else {

					if ActiveRequest.OnTick != nil {
						ActiveRequest.OnTick(ActiveRequest, tick)
					}

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
		select {
		case hail := <-cmn.HailChannel:

			log.Trace().Msgf("Hail received: %s", hail.Title)

			if hail.TimeoutSec == 0 {
				hail.TimeoutSec = cmn.Config.TimeoutSec
			}
			on_top := add(hail)
			if on_top {
				HailPane.open(hail)
			}

		case hail := <-cmn.RemoveHailChannel:
			log.Trace().Msgf("ProcessHails: Remove hail received: %s", hail.Title)
			remove(hail)
		}
	}
}

func (p *HailPaneType) open(hail *cmn.HailRequest) {

	log.Trace().Msgf("HailPane: open: %s", hail.Title)

	if ActiveRequest != hail {
		if ActiveRequest != nil {
			if ActiveRequest.OnSuspend != nil {
				ActiveRequest.OnSuspend(hail)
			}
			ActiveRequest.Suspended = true
		}
	}

	ActiveRequest = hail

	if hail.Suspended {
		if hail.OnResume != nil {
			hail.OnResume(hail)
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
		hail.OnOpen(hail, Gui, HailPane.View)
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
			log.Error().Err(err).Msgf("SetView error: %s", err)
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

		p.View.OnClickHotspot = func(v *gocui.View, hs *gocui.Hotspot) {
			if ActiveRequest == nil {
				return
			}

			if hs != nil {
				switch strings.ToLower(hs.Value) {
				case "button ok":
					log.Trace().Msgf("HailPane: button Ok")
					if ActiveRequest.OnOk != nil {
						ActiveRequest.OnOk(ActiveRequest)
					}
					remove(ActiveRequest)
				case "button cancel":
					log.Trace().Msgf("HailPane: button Cancel")
					if ActiveRequest.OnCancel != nil {
						ActiveRequest.OnCancel(ActiveRequest)
					}
					remove(ActiveRequest)
				default:
					if ActiveRequest.OnClickHotspot != nil {
						ActiveRequest.OnClickHotspot(ActiveRequest, v, hs)
					}
				}
			}
		}
		p.View.OnOverHotspot = func(v *gocui.View, hs *gocui.Hotspot) {
			if ActiveRequest == nil {
				return
			}

			if ActiveRequest.OnOverHotspot != nil {
				ActiveRequest.OnOverHotspot(ActiveRequest, v, hs)
			}
		}
	}
}
