package ui

import (
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
)

func DlgWaletOpen(name string) *gocui.Popup {
	return &gocui.Popup{
		Title: "Open Wallet " + name,
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {

			if hs != nil {
				switch hs.Value {
				case "button Ok":
					pass := v.GetInput("pass")

					// Close dialog immediately and open wallet in background
					Gui.HidePopup()
					Printf("\nWallet is opening...\n")

					go func() {
						err := cmn.Open(name, pass)

						if err != nil {
							Notification.ShowErrorf("Error opening wallet: %s", err)
							Printf("Error opening wallet: %s\n", err)
							return
						}

						Notification.Showf("Wallet %s open", name)
						Printf("Wallet %s opened\n", name)
						Terminal.SetCommandPrefix(name)
					}()

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
