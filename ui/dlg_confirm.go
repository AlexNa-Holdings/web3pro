package ui

import (
	"github.com/AlexNa-Holdings/web3pro/gocui"
)

func DlgConfirm(title string, text string, action func()) *gocui.Popup {
	return &gocui.Popup{
		Title: title,
		OnOpen: func(v *gocui.View) {
			v.SetFocus(1) // focus on cancel button
		},
		OnClose: func(v *gocui.View) {
			Gui.SetCurrentView("terminal.input")
		},
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {

			if hs != nil {
				switch hs.Value {
				case "button Ok":
					if action != nil {
						action()
					}
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: text + `
<c><button text:Ok tip:"create wallet">  <button text:Cancel>`,
	}
}
