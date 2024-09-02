package ui

import (
	"sync"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

var popupQueue []*bus.Message
var popupQueueM sync.Mutex

func processPopup(m *bus.Message) {
	if m.TimerID != 0 {
		res := bus.Fetch("timer", "pause", m.TimerID)
		if res.Error != nil {
			log.Error().Err(res.Error).Msg("processPopup: pause timer")
			return
		}
	}

	if Gui.GetCurentPopup() == nil {
		showPopup(m)
	} else {
		popupQueueM.Lock()
		popupQueue = append(popupQueue, m)
		popupQueueM.Unlock()
	}
}

func showPopup(m *bus.Message) {
	popup, ok := m.Data.(*gocui.Popup)
	if !ok {
		log.Error().Msgf("processPopup: invalid data: %v", m.Data)
	}

	onClose := popup.OnClose
	popup.OnClose = func(v *gocui.View) {
		if m.TimerID != 0 {
			res := bus.Fetch("timer", "resume", m.TimerID)
			if res.Error != nil {
				log.Error().Err(res.Error).Msg("processPopup: resume timer")
			}
		}

		if onClose != nil {
			onClose(v)
		}

		m.Respond("OK", nil)

		popupQueueM.Lock()
		if len(popupQueue) > 0 {
			nm := popupQueue[0]
			popupQueue = popupQueue[1:]
			popupQueueM.Unlock()
			showPopup(nm)
		} else {
			popupQueueM.Unlock()
			Gui.SetCurrentView("terminal.input")
		}
	}
	Gui.ShowPopup(popup)
}
