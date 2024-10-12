package ui

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
)

func DlgLP_V3_add(b *cmn.Blockchain, address string, name string, url string) *gocui.Popup {
	template := fmt.Sprintf(`
   Chain: %s	
 Address: <input id:address size:32 value:"">
    Name: <input id:name size:32 value:""> 
     URL: <input id:url size:32 value:"">

<c><button text:Ok tip:"create wallet">  <button text:Cancel>`, b.Name)

	return &gocui.Popup{
		Title: "Add LP v3",
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
			v.SetInput("address", address)
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

					addr := v.GetInput("address")
					a := common.HexToAddress(addr)

					if a == (common.Address{}) {
						Notification.ShowError("Invalid address")
						break
					}

					w := cmn.CurrentWallet
					if w == nil {
						Notification.ShowError("No wallet open")
						break
					}

					lp := w.GetLP_V3(b.ChainId, a)
					if lp != nil {
						Notification.ShowError("Provider already exists")
						break
					}

					url := v.GetInput("url")

					err := w.AddLP_V3(&cmn.LP_V3{
						Name:     name,
						Provider: a,
						ChainId:  b.ChainId,
						URL:      url,
					})

					if err != nil {
						Notification.ShowErrorf("Error adding LP: %s", err)
						break
					}
					Notification.Showf("LP v3 %s created", name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: template,
	}
}
