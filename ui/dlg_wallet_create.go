package ui

import (
	"github.com/AlexNa-Holdings/web3pro/gocui"
)

func DlgWaletCreate() *gocui.Popup {
	return &gocui.Popup{
		Title: "Create Wallet",
		// Subtitle: "Enter wallet name",
		OnOverHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				Bottom.Printf(hs.Tip)
			} else {
				Bottom.Printf("")
			}
		},
		OnClose: func(v *gocui.View) {
			Gui.SetCurrentView("terminal.input")
		},
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {

			if hs != nil {
				switch hs.Value {
				case "button Ok":
				// Create wallet
				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: `
      Name: <i id:name size:24 value:"tak tak tak">
  Password: <i id:pass masked:true size:24>
  (Retype): <i id:retype masked:true size:24>

<c><b text:Ok tip:"ha ha ha">  <b text:Cancel>`,
	}
}
