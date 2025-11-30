package ui

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
)

func DlgLP_V4_edit(b *cmn.Blockchain, provider string, name string, url string, subgraphURL string) *gocui.Popup {
	template := fmt.Sprintf(`
       Chain: %s
    Provider: %s
        Name: <input id:name size:32 value:"">
         URL: <input id:url size:42 value:"">
 SubgraphURL: <input id:subgraph_url size:42 value:"">

<c><button text:Ok tip:"save changes">  <button text:Cancel>`, b.Name, provider)

	return &gocui.Popup{
		Title: "Edit LP v4",
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
			v.SetInput("subgraph_url", subgraphURL)
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

					a := common.HexToAddress(provider)

					lp := w.GetLP_V4(b.ChainId, a)
					if lp == nil {
						Notification.ShowError("Provider not found")
						break
					}

					url := v.GetInput("url")
					subgraphURL := v.GetInput("subgraph_url")

					lp.Name = name
					lp.URL = url
					lp.SubgraphURL = subgraphURL

					err := cmn.CurrentWallet.Save()
					if err != nil {
						Notification.ShowErrorf("Error updating LP: %s", err)
						break
					}
					Notification.Showf("LP v4 %s changed", name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: template,
	}
}
