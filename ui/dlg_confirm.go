package ui

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/gocui"
)

func DlgConfirm(title string, text string, action func() bool) *gocui.Popup {
	return &gocui.Popup{
		Title: title,
		OnOpen: func(v *gocui.View) {
			v.SetFocus(1) // focus on cancel button
		},
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {

			if hs != nil {
				switch strings.ToLower(hs.Value) {
				case "button ok":
					if action != nil {
						if action() {
							Gui.HidePopup()
						}
					} else {
						Gui.HidePopup()
					}

					Flush()

				case "button cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: text + `
<c><button text:Ok tip:"create wallet">  <button text:Cancel>`,
	}
}
