package ui

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

type HailPaneType struct {
	PaneDescriptor
}

var HailPane HailPaneType = HailPaneType{
	PaneDescriptor: PaneDescriptor{
		MinWidth:    40,
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

func (p *HailPaneType) SetView(x0, y0, x1, y1 int, overlap byte) {
	var err error

	if ActiveRequest == nil {
		return
	}

	active_hail := ActiveRequest.Data.(*bus.B_Hail)

	v, err := Gui.SetView("hail", x0, y0, x1, y1, overlap)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}

		p.PaneDescriptor.View = v
		v.FrameRunes = RUNES
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
				active_hail := ActiveRequest.Data.(*bus.B_Hail)
				if active_hail.Template != "" {
					v.RenderTemplate(active_hail.Template)
				}
			}
		}

		v.OnResize(v) // render template

		v.OnClickTitle = func(v *gocui.View) { // reset timer
			bus.Send("timer", "reset", ActiveRequest.TimerID)
			Gui.UpdateAsync(func(g *gocui.Gui) error {
				HailPane.UpdateSubtitle(nil)
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
					close := true
					if active_hail.OnOk != nil {
						close = active_hail.OnOk(ActiveRequest)
					}
					if close {
						remove(ActiveRequest)
					}
				case "button cancel":
					log.Trace().Msgf("HailPane: button Cancel")
					if active_hail.OnCancel != nil {
						active_hail.OnCancel(ActiveRequest)
					}
					remove(ActiveRequest)
				default:
					if active_hail.OnClickHotspot != nil {
						go active_hail.OnClickHotspot(ActiveRequest, v, hs)
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
				active_hail.OnOverHotspot(ActiveRequest, v, hs)
			}
		}

		if active_hail.OnOpen != nil {
			active_hail.OnOpen(ActiveRequest, Gui, v)
		}

	}
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

	log.Debug().Msgf("hqAdd: hail %s  on_top: %v", m.Data.(*bus.B_Hail).Title, on_top)

	HQMutex.Lock()
	defer HQMutex.Unlock()

	if on_top {
		HailQueue = append([]*bus.Message{m}, HailQueue...)
	} else {
		HailQueue = append(HailQueue, m)
	}
}

func hqRemove(m *bus.Message) {
	HQMutex.Lock()
	defer HQMutex.Unlock()

	for i := 0; i < len(HailQueue); i++ {
		if HailQueue[i] == m {
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

	if m == nil {
		log.Error().Msg("HailPane: remove: nil message")
		return
	}

	if m.Data == nil {
		log.Error().Msg("HailPane: remove: nil data")
		return
	}

	log.Debug().Msgf("HailPane: remove: %s", m.Data.(*bus.B_Hail).Title)

	hail := m.Data.(*bus.B_Hail)
	log.Trace().Msgf("Removing hail %s", hail.Title)

	hqRemove(m)

	if hail.OnClose != nil {
		hail.OnClose(m)
	}

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
		hail.OnCancel(m)
	}

	remove(m)
}

func (p *HailPaneType) open(m *bus.Message) {

	if m == ActiveRequest {
		log.Error().Msg("HailPane: open: already active")
		return
	}

	hail := m.Data.(*bus.B_Hail)
	log.Debug().Msgf("HailPane: open: %s on_top: %v", hail.Title, hail.Priorized)
	bus.Send("sound", "play", nil)

	if ActiveRequest != nil {
		active_hail := ActiveRequest.Data.(*bus.B_Hail)
		if active_hail.OnSuspend != nil {
			active_hail.OnSuspend(ActiveRequest)
		}
		active_hail.Suspended = true
	}

	ActiveRequest = m

	if HailPane.View == nil {
		HailPane.PaneDescriptor.ShowPane()
	}

	if HailPane.View != nil && hail.Template != "" {
		HailPane.View.RenderTemplate(hail.Template)
	}

	if hail.Suspended {
		if hail.OnResume != nil {
			hail.OnResume(m)
		}
		hail.Suspended = false
	} else {
		if hail.OnOpen != nil {
			hail.OnOpen(m, Gui, HailPane.View)
		}
	}

	Gui.UpdateAsync(func(g *gocui.Gui) error {
		HailPane.UpdateSubtitle(nil)
		return nil
	})

}

func (p *HailPaneType) UpdateSubtitle(left_map map[int]time.Duration) {
	if HailPane.View != nil && ActiveRequest != nil {
		left_s := ""
		sec := int(time.Duration(0))

		if left_map != nil {

			if d, ok := left_map[ActiveRequest.TimerID]; ok {
				sec = int(d.Seconds() + 0.5)
			}
		} else {
			res := bus.Fetch("timer", "left", ActiveRequest.TimerID)
			if res.Error != nil {
				log.Error().Err(res.Error).Msg("Error fetching timer left")
				return
			} else {
				sec = int(res.Data.(time.Duration).Seconds() + 0.5)
			}
		}

		if sec < 10 {
			p.View.SubTitleBgColor = Theme.ErrorFgColor
		} else {
			p.View.SubTitleBgColor = Theme.HelpBgColor
		}

		if sec > 0 {
			left_s = fmt.Sprintf("%d", sec)
		}

		if len(HailQueue) > 1 {
			p.View.Subtitle = fmt.Sprintf("(%d) %s", len(HailQueue), left_s)

			// DEBUG
			for i, m := range HailQueue {
				log.Debug().Msgf("HailQueue[%d]: %s", i, m.Data.(*bus.B_Hail).Title)
			}

		} else {
			p.View.Subtitle = left_s
		}

		HailPane.View.Subtitle = p.View.Subtitle
	}
}
