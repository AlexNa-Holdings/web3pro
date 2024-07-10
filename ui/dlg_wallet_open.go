package ui

import (
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
)

func DlgWaletOpen(name string) *gocui.Popup {
	return &gocui.Popup{
		Title: "Open Wallet " + name,
		OnClose: func(v *gocui.View) {
			Gui.SetCurrentView("terminal.input")
		},
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {

			if hs != nil {
				switch hs.Value {
				case "button Ok":
					pass := v.GetInput("pass")

					err := cmn.Open(name, pass)

					if err != nil {
						Notification.ShowErrorf("Error opening wallet: %s", err)
						v.SetInput("pass", "")
						v.SetFocus(0)
						break
					}

					Notification.Showf("Wallet %s open", name)
					Terminal.SetCommandPrefix(name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: `
 Password: <input id:pass masked:true size:24> 
 <c>
 <button text:Ok tip:"create wallet">  <button text:Cancel>`,
	}
}
