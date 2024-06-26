package ui

import (
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
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
					name := v.GetInput("name")

					if len(name) == 0 {
						Notification.ShowError("Name cannot be empty")
						break
					}

					if wallet.Exists(name) {
						Notification.ShowErrorf("Wallet %s already exists", name)
						break
					}

					pass := v.GetInput("pass")
					retype := v.GetInput("retype")

					if pass != retype {
						Notification.ShowError("Passwords do not match")
						break
					}

					err := wallet.Create(name, pass)

					if err != nil {
						Notification.ShowErrorf("Error creating wallet: %s", err)
						break
					}

					Notification.Showf("Wallet %s created", name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: `
      Name: <i id:name size:24 value:"">
  Password: <i id:pass masked:true size:24>
  (Retype): <i id:retype masked:true size:24>
<c>
<button text:Ok tip:"create wallet">  <button text:Cancel>`,
	}
}
