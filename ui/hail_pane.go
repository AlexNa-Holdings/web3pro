package ui

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/AlexNa-Holdings/web3pro/bus"
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
var HQMutex = &sync.Mutex{}

func add(m *bus.Message) bool { // returns if on top

	hail, ok := m.Data.(*bus.B_Hail)
	if !ok {
		log.Error().Msg("Hail data is not of type HailRequest")
		return false
	}

	log.Trace().Msgf("Adding hail: %s", hail.Title)
	hqAdd(m, hail.Priorized)
	return m == HailQueue[0]
}

func hqAdd(m *bus.Message, on_top bool) {
	HQMutex.Lock()
	defer HQMutex.Unlock()

	if on_top {
		HailQueue = append([]*bus.Message{m}, HailQueue...)
	} else {
		HailQueue = append(HailQueue, m)
	}
}

func hqRemove(h *bus.B_Hail) {
	HQMutex.Lock()
	defer HQMutex.Unlock()

	for i := 0; i < len(HailQueue); i++ {
		if HailQueue[i].Data == h {
			HailQueue = append(HailQueue[:i], HailQueue[i+1:]...)
			i -= 1
		}
	}
}

func hqGetTop() *bus.Message {
	HQMutex.Lock()
	defer HQMutex.Unlock()

	if len(HailQueue) > 0 {
		return HailQueue[0]
	}
	return nil
}

func remove(m *bus.Message) {
	hail := m.Data.(*bus.B_Hail)
	log.Trace().Msgf("Removing hail %s", hail.Title)

	hqRemove(hail)

	if hail.OnClose != nil {
		hail.OnClose(hail)
	}

	bus.Send("timer", "delete", &bus.B_TimerDelete{ID: m.TimerID})
	m.Respond(nil, nil)

	if ActiveRequest != nil && ActiveRequest.Data == hail {
		ActiveRequest = nil

		top := hqGetTop()
		if top != nil {
			HailPane.open(top)
		} else {
			Gui.DeleteView("hail")
			HailPane.View = nil
			Flush()
		}
	}
}

func cancel(m *bus.Message) {
	hail := m.Data.(*bus.B_Hail)

	if hail.OnCancel != nil {
		hail.OnCancel(hail)
	}

	remove(m)
}

func (p *HailPaneType) open(m *bus.Message) {
	hail := m.Data.(*bus.B_Hail)
	log.Trace().Msgf("HailPane: open: %s", hail.Title)

	if ActiveRequest != nil {
		if ActiveRequest.Data != hail {
			active_hail := ActiveRequest.Data.(*bus.B_Hail)
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

		left := int(bus.GetTimeLeft(ActiveRequest.TimerID).Seconds())

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

	active_hail := ActiveRequest.Data.(*bus.B_Hail)

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
			bus.Send("timer", "reset", &bus.B_TimerReset{ID: ActiveRequest.TimerID})
			Gui.UpdateAsync(func(g *gocui.Gui) error {
				HailPane.UpdateSubtitle()
				return nil
			})
		}

		p.View.OnClickHotspot = func(v *gocui.View, hs *gocui.Hotspot) {
			if ActiveRequest == nil {
				return
			}

			active_hail := ActiveRequest.Data.(*bus.B_Hail)

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

			active_hail := ActiveRequest.Data.(*bus.B_Hail)

			if active_hail.OnOverHotspot != nil {
				active_hail.OnOverHotspot(active_hail, v, hs)
			}
		}
	}
}
