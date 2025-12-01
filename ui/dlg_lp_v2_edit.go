package ui

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
)

func DlgLP_V2_edit(b *cmn.Blockchain, factory string, name string, url string, subgraphID string) *gocui.Popup {
	template := fmt.Sprintf(`
     Chain: %s
   Factory: %s
      Name: <input id:name size:32 value:"">
       URL: <input id:url size:42 value:"">
SubgraphID: <input id:subgraph size:48 value:"">

<c><button text:Ok tip:"save changes">  <button text:Cancel>`, b.Name, factory)

	return &gocui.Popup{
		Title: "Edit LP v2",
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
			v.SetInput("subgraph", subgraphID)
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

					f := common.HexToAddress(factory)

					lp := w.GetLP_V2(b.ChainId, f)
					if lp == nil {
						Notification.ShowError("Provider not found")
						break
					}

					url := v.GetInput("url")
					subgraph := v.GetInput("subgraph")

					lp.Name = name
					lp.URL = url
					lp.SubgraphID = subgraph

					err := cmn.CurrentWallet.Save()
					if err != nil {
						Notification.ShowErrorf("Error updating LP: %s", err)
						break
					}
					Notification.Showf("LP v2 %s changed", name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: template,
	}
}
