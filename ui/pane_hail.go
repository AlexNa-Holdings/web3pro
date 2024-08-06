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
	PaneDescriptor
}

var HailPane HailPaneType = HailPaneType{
	PaneDescriptor: PaneDescriptor{
		MinWidth:    33,
		MinHeight:   1,
		fixed_width: true,
	},
}

var ActiveRequest *bus.Message
var HailQueue []*bus.Message
var HQMutex = &sync.Mutex{}

func (p *HailPaneType) GetDesc() *PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *HailPaneType) GetTemplate() string {
	if ActiveRequest != nil {
		return ActiveRequest.Data.(*bus.B_Hail).Template
	}
	return ""
}

func (p *HailPaneType) SetView(x0, y0, x1, y1 int) {
	var err error

	if ActiveRequest == nil {
		return
	}

	active_hail := ActiveRequest.Data.(*bus.B_Hail)

	v, err := Gui.SetView("hail", x0, y0, x1, y1, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}

		v.Title = "Hail"
		if active_hail.Title != "" {
			v.Title = active_hail.Title
		}
		v.SubTitleFgColor = Theme.HelpFgColor
		v.SubTitleBgColor = Theme.HelpBgColor
		v.FrameColor = Gui.ActionBgColor
		v.TitleColor = Gui.ActionFgColor
		v.EmFgColor = Gui.ActionBgColor
		v.ScrollBar = true
		v.OnResize = func(v *gocui.View) {
			if ActiveRequest != nil {
				if active_hail.Template != "" {
					v.RenderTemplate(active_hail.Template)
				}
			}
		}

		v.OnResize(v) // render template

		v.OnClickTitle = func(v *gocui.View) { // reset timer
			bus.Send("timer", "reset", &bus.B_TimerReset{ID: ActiveRequest.TimerID})
			Gui.UpdateAsync(func(g *gocui.Gui) error {
				HailPane.UpdateSubtitle()
				return nil
			})
		}

		v.OnClickHotspot = func(v *gocui.View, hs *gocui.Hotspot) {
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
		v.OnOverHotspot = func(v *gocui.View, hs *gocui.Hotspot) {
			if ActiveRequest == nil {
				return
			}

			active_hail := ActiveRequest.Data.(*bus.B_Hail)

			if active_hail.OnOverHotspot != nil {
				active_hail.OnOverHotspot(active_hail, v, hs)
			}
		}
	}
	p.PaneDescriptor.View = v
}

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
			HailPane.PaneDescriptor.HidePane()
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

	if HailPane.View == nil {
		HailPane.PaneDescriptor.ShowPane()
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
			p.View.SubTitleBgColor = Theme.ErrorFgColor
		} else {
			p.View.SubTitleBgColor = Theme.HelpBgColor
		}

		if len(HailQueue) > 1 {
			p.View.Subtitle = fmt.Sprintf("(%d) %d", len(HailQueue), left)
		} else {
			p.View.Subtitle = fmt.Sprintf("%d", left)
		}

		HailPane.View.Subtitle = p.View.Subtitle
	}
}
