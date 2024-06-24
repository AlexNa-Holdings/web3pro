package ui

import (
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
)

func DlgAddressAdd(signer string, path string) *gocui.Popup {
	template := fmt.Sprintf(`
   Name: <i id:name size:32 value:""> 
 Signer: %s
   Path: %s

<c><b text:Ok tip:"create wallet">  <b text:Cancel>`, signer, path)

	return &gocui.Popup{
		Title: "Add address",
		// Subtitle: "Enter wallet name",
		OnOverHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				Bottom.Printf(hs.Tip)
			} else {
				Bottom.Printf("")
			}
		},
		OnOpen: func(v *gocui.View) {

			name := ""

			lastSlashIndex := strings.LastIndex(path, "/")
			if lastSlashIndex >= 0 {

				name = signer + "_" + path[lastSlashIndex+1:]
			}

			v.SetInput("name", name)
		},
		OnClose: func(v *gocui.View) {
			Gui.SetCurrentView("terminal.input")
		},
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {

			if hs != nil {
				switch hs.Value {
				case "button Ok":
					err := wallet.CurrentWallet.Save()
					if err != nil {
						Notification.ShowErrorf("Error creating signer: %s", err)
						break
					}

					// Notification.Showf("Signer %s created", name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: template,
	}
}
