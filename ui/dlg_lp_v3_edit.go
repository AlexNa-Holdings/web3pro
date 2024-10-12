package ui

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
)

func DlgLP_V3_edit(b *cmn.Blockchain, address string, name string, url string) *gocui.Popup {
	template := fmt.Sprintf(`
   Chain: %s	
 Address: %s
    Name: <input id:name size:32 value:""> 
     URL: <input id:url size:32 value:"">

<c><button text:Ok tip:"create wallet">  <button text:Cancel>`, b.Name, address)

	return &gocui.Popup{
		Title: "Edit LP v3",
		// Subtitle: "Enter wallet name",
		OnOverHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				Bottom.Printf(hs.Tip)
			} else {
				Bottom.Printf("")
			}
		},
		OnOpen: func(v *gocui.View) {
			v.SetInput("name", name)
			v.SetInput("url", url)
		},
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {

			if hs != nil {
				switch hs.Value {
				case "button Ok":
					name := v.GetInput("name")

					if name == "" {
						Notification.ShowError("Name cannot be empty")
						break
					}

					w := cmn.CurrentWallet
					if w == nil {
						Notification.ShowError("No wallet open")
						break
					}

					a := common.HexToAddress(address)

					lp := w.GetLP_V3(b.ChainId, a)
					if lp == nil {
						Notification.ShowError("Provider not found")
						break
					}

					url := v.GetInput("url")

					lp.Name = name
					lp.URL = url

					err := cmn.CurrentWallet.Save()
					if err != nil {
						Notification.ShowErrorf("Error updating LP: %s", err)
						break
					}
					Notification.Showf("LP v3 %s changed", name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: template,
	}
}
