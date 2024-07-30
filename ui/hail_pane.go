package ui

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/AlexNa-Holdings/web3pro/bus"
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

var ActiveRequest *bus.Message
var HailQueue []*bus.Message
var Mutex = &sync.Mutex{}

func add(m *bus.Message) bool { // returns if on top

	hail, ok := m.Data.(*cmn.HailRequest)
	if !ok {
		log.Error().Msg("Hail data is not of type HailRequest")
		return false
	}

	log.Trace().Msgf("Adding hail: %s", hail.Title)
	Mutex.Lock()
	defer Mutex.Unlock()

	if hail.Priorized {
		HailQueue = append([]*bus.Message{m}, HailQueue...)
	} else {
		HailQueue = append(HailQueue, m)
	}
	return m == HailQueue[0]
}

func remove(m *bus.Message) {
	hail := m.Data.(*cmn.HailRequest)
	log.Trace().Msgf("Removing hail %s", hail.Title)

	Mutex.Lock()
	for i, h := range HailQueue {
		if h.Data == hail {
			HailQueue = append(HailQueue[:i], HailQueue[i+1:]...)
		}
	}
	Mutex.Unlock()

	if hail.OnClose != nil {
		hail.OnClose(hail)
	}

	bus.Send("timer", "remove", &bus.BM_TimerDone{ID: m.TimerID})
	m.Respond("OK", nil)

	if ActiveRequest != nil && ActiveRequest.Data == hail {
		ActiveRequest = nil

		if len(HailQueue) > 0 {
			HailPane.open(HailQueue[0])
		} else {
			Gui.DeleteView("hail")
			HailPane.View = nil
			Flush()
		}
	}
}

func cancel(m *bus.Message) {
	hail := m.Data.(*cmn.HailRequest)

	if hail.OnCancel != nil {
		hail.OnCancel(hail)
	}

	remove(m)
}

func ProcessHails() {

	log.Debug().Msg("ProcessHails")

	ch := bus.Subscribe("ui", "timer")
	defer bus.Unsubscribe(ch)

	for msg := range ch {
		switch msg.Type {
		case "hail":
			log.Trace().Msg("ProcessHails: Hail received")
			if hail, ok := msg.Data.(*cmn.HailRequest); ok {
				log.Trace().Msgf("Hail received: %s", hail.Title)

				if hail.TimeoutSec == 0 {
					hail.TimeoutSec = cmn.Config.BusTimeout
				}
				if on_top := add(msg); on_top {
					HailPane.open(msg)
				}
			}
		case "remove_hail":
			if hail, ok := msg.Data.(*cmn.HailRequest); ok {
				log.Trace().Msgf("ProcessHails: Remove hail received: %s", hail.Title)

				var m *bus.Message
				Mutex.Lock()
				for i, h := range HailQueue {
					if h.Data == hail {
						m = HailQueue[i]
						break
					}
				}
				Mutex.Unlock()

				if m != nil {
					remove(m)
				}
			}
		case "tick":
			if _, ok := msg.Data.(*bus.BM_TimerTick); ok {
				if ActiveRequest != nil {
					Gui.UpdateAsync(func(g *gocui.Gui) error {
						HailPane.UpdateSubtitle()
						return nil
					})
				}
			}
		case "done":
			if d, ok := msg.Data.(*bus.BM_TimerDone); ok {
				log.Trace().Msgf("Alert: %v", d.ID)
				if ActiveRequest != nil {
					if ActiveRequest.TimerID == d.ID {
						cancel(ActiveRequest)
					}
				}
			}
		}
	}
}

func (p *HailPaneType) open(m *bus.Message) {
	hail := m.Data.(*cmn.HailRequest)
	log.Trace().Msgf("HailPane: open: %s", hail.Title)

	if ActiveRequest != nil {
		if ActiveRequest.Data != hail {
			active_hail := ActiveRequest.Data.(*cmn.HailRequest)
			if active_hail.OnSuspend != nil {
				active_hail.OnSuspend(hail)
			}
			active_hail.Suspended = true
		}
	}

	ActiveRequest = m
	if hail.Suspended {
		if hail.OnResume != nil {
			hail.OnResume(hail)
		}
		hail.Suspended = false
	}

	maxX, _ := Gui.Size()
	n_lines := gocui.EstimateTemplateLines(hail.Template, maxX/2)
	HailPane.MinHeight = n_lines + 2

	if HailPane.View == nil {
		HailPane.SetView(Gui, 0, 0, maxX/2, n_lines)
	} else {
		if hail.Template != "" {
			HailPane.View.RenderTemplate(hail.Template)
			Flush()
		}
	}

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

		left := bus.GetTimerSecondsLeft(ActiveRequest.TimerID)

		if left < 10 {
			p.SubTitleBgColor = Theme.ErrorFgColor
		} else {
			p.SubTitleBgColor = Theme.HelpBgColor
		}

		if len(HailQueue) > 1 {
			p.Subtitle = fmt.Sprintf("(%d) %d", len(HailQueue), left)
		} else {
			p.Subtitle = fmt.Sprintf("%d", left)
		}

		HailPane.View.Subtitle = p.Subtitle
	}
}

func (p *HailPaneType) SetView(g *gocui.Gui, x0, y0, x1, y1 int) {
	var err error

	if ActiveRequest == nil {
		return
	}

	active_hail := ActiveRequest.Data.(*cmn.HailRequest)

	if p.View, err = g.SetView("hail", x0, y0, x1, y1, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}

		p.View.Title = "Hail"
		if active_hail.Title != "" {
			p.View.Title = active_hail.Title
		}
		p.View.SubTitleFgColor = Theme.HelpFgColor
		p.View.SubTitleBgColor = Theme.HelpBgColor
		p.View.FrameColor = Gui.ActionBgColor
		p.View.TitleColor = Gui.ActionFgColor
		p.View.EmFgColor = Gui.ActionBgColor
		p.View.ScrollBar = true
		p.View.OnResize = func(v *gocui.View) {
			if ActiveRequest != nil {
				if active_hail.Template != "" {
					v.RenderTemplate(active_hail.Template)
				}
			}
		}
		p.View.OnClickTitle = func(v *gocui.View) { // reset timer
			bus.Send("timer", "reset", &bus.BM_TimerReset{ID: ActiveRequest.TimerID})
			Gui.UpdateAsync(func(g *gocui.Gui) error {
				HailPane.UpdateSubtitle()
				return nil
			})
		}

		p.View.OnClickHotspot = func(v *gocui.View, hs *gocui.Hotspot) {
			if ActiveRequest == nil {
				return
			}

			active_hail := ActiveRequest.Data.(*cmn.HailRequest)

			if hs != nil {
				switch strings.ToLower(hs.Value) {
				case "button ok":
					log.Trace().Msgf("HailPane: button Ok")
					if active_hail.OnOk != nil {
						active_hail.OnOk(active_hail)
					}
					remove(ActiveRequest)
				case "button cancel":
					log.Trace().Msgf("HailPane: button Cancel")
					if active_hail.OnCancel != nil {
						active_hail.OnCancel(active_hail)
					}
					remove(ActiveRequest)
				default:
					if active_hail.OnClickHotspot != nil {
						active_hail.OnClickHotspot(active_hail, v, hs)
					}
				}
			}
		}
		p.View.OnOverHotspot = func(v *gocui.View, hs *gocui.Hotspot) {
			if ActiveRequest == nil {
				return
			}

			active_hail := ActiveRequest.Data.(*cmn.HailRequest)

			if active_hail.OnOverHotspot != nil {
				active_hail.OnOverHotspot(active_hail, v, hs)
			}
		}
	}
}
