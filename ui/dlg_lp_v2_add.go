package ui

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
)

func DlgLP_V2_add(b *cmn.Blockchain, factory string, router string, name string, url string, subgraphID string) *gocui.Popup {
	template := fmt.Sprintf(`
     Chain: %s
   Factory: <input id:factory size:42 value:"">
    Router: <input id:router size:42 value:"">
      Name: <input id:name size:32 value:"">
       URL: <input id:url size:42 value:"">
SubgraphID: <input id:subgraph size:48 value:"">

<c><button text:Ok tip:"add provider">  <button text:Cancel>`, b.Name)

	return &gocui.Popup{
		Title: "Add LP v2",
		OnOverHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				Bottom.Printf(hs.Tip)
			} else {
				Bottom.Printf("")
			}
		},
		OnOpen: func(v *gocui.View) {
			v.SetInput("factory", factory)
			v.SetInput("router", router)
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

					factoryAddr := v.GetInput("factory")
					f := common.HexToAddress(factoryAddr)

					if f == (common.Address{}) {
						Notification.ShowError("Invalid factory address")
						break
					}

					routerAddr := v.GetInput("router")
					r := common.HexToAddress(routerAddr)

					if r == (common.Address{}) {
						Notification.ShowError("Invalid router address")
						break
					}

					w := cmn.CurrentWallet
					if w == nil {
						Notification.ShowError("No wallet open")
						break
					}

					lp := w.GetLP_V2(b.ChainId, f)
					if lp != nil {
						Notification.ShowError("Provider already exists")
						break
					}

					url := v.GetInput("url")
					subgraph := v.GetInput("subgraph")

					err := w.AddLP_V2(&cmn.LP_V2{
						Name:       name,
						Factory:    f,
						Router:     r,
						ChainId:    b.ChainId,
						URL:        url,
						SubgraphID: subgraph,
					})

					if err != nil {
						Notification.ShowErrorf("Error adding LP: %s", err)
						break
					}
					Notification.Showf("LP v2 %s created", name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: template,
	}
}
