package ui

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
)

func DlgLP_V4_add(b *cmn.Blockchain, provider string, poolManager string, stateView string, name string, url string, subgraphURL string) *gocui.Popup {
	template := fmt.Sprintf(`
       Chain: %s
    Provider: <input id:provider size:42 value:"">
 PoolManager: <input id:pool_manager size:42 value:"">
   StateView: <input id:state_view size:42 value:"">
        Name: <input id:name size:32 value:"">
         URL: <input id:url size:42 value:"">
 SubgraphURL: <input id:subgraph_url size:42 value:"">

<c><button text:Ok tip:"add provider">  <button text:Cancel>`, b.Name)

	return &gocui.Popup{
		Title: "Add LP v4",
		OnOverHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				Bottom.Printf(hs.Tip)
			} else {
				Bottom.Printf("")
			}
		},
		OnOpen: func(v *gocui.View) {
			v.SetInput("name", name)
			v.SetInput("provider", provider)
			v.SetInput("pool_manager", poolManager)
			v.SetInput("state_view", stateView)
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

					providerAddr := v.GetInput("provider")
					p := common.HexToAddress(providerAddr)

					if p == (common.Address{}) {
						Notification.ShowError("Invalid provider address")
						break
					}

					poolManagerAddr := v.GetInput("pool_manager")
					pm := common.HexToAddress(poolManagerAddr)

					if pm == (common.Address{}) {
						Notification.ShowError("Invalid pool manager address")
						break
					}

					stateViewAddr := v.GetInput("state_view")
					sv := common.HexToAddress(stateViewAddr)
					// StateView can be empty (zero address) for chains without it

					w := cmn.CurrentWallet
					if w == nil {
						Notification.ShowError("No wallet open")
						break
					}

					lp := w.GetLP_V4(b.ChainId, p)
					if lp != nil {
						Notification.ShowError("Provider already exists")
						break
					}

					url := v.GetInput("url")
					subgraphURL := v.GetInput("subgraph_url")

					err := w.AddLP_V4(&cmn.LP_V4{
						Name:        name,
						Provider:    p,
						PoolManager: pm,
						StateView:   sv,
						ChainId:     b.ChainId,
						URL:         url,
						SubgraphURL: subgraphURL,
					})

					if err != nil {
						Notification.ShowErrorf("Error adding LP: %s", err)
						break
					}
					Notification.Showf("LP v4 %s created", name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: template,
	}
}
