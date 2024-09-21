package ui

import (
	"strconv"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
)

// name == ""  mreans add new custom blockchain
func DlgBlockchain(name string) *gocui.Popup {

	if cmn.CurrentWallet == nil {
		Notification.ShowError("No wallet open")
		return nil
	}

	bch_index := -1

	if name != "" {
		for i, bch := range cmn.CurrentWallet.Blockchains {
			if bch.Name == name {
				bch_index = i
				break
			}
		}

		if bch_index == -1 {
			Notification.ShowErrorf("Blockchain %s not found", name)
			return nil
		}
	}

	title := "Add Blockchain"
	if name != "" {
		title = "Edit Blockchain"
	}

	return &gocui.Popup{
		Title: title,
		OnOverHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				Bottom.Printf(hs.Tip)
			} else {
				Bottom.Printf("")
			}
		},
		OnOpen: func(v *gocui.View) {
			v.SetList("explorer_api_type", cmn.EXPLORER_API_TYPES)
			if bch_index != -1 {
				bch := cmn.CurrentWallet.Blockchains[bch_index]
				v.SetInput("name", bch.Name)
				v.SetInput("rpc", bch.Url)
				v.SetInput("chainid", strconv.Itoa(int(bch.ChainID)))
				v.SetInput("explorer", bch.ExplorerUrl)
				v.SetInput("api_token", bch.ExplorerAPIToken)
				v.SetInput("currency", bch.Currency)
				if bch.WTokenAddress != (common.Address{}) {
					v.SetInput("wtoken_address", bch.WTokenAddress.String())
				}
				v.SetInput("explorer_api_url", bch.ExplorerAPIUrl)
			}
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

					for i, bch := range cmn.CurrentWallet.Blockchains {
						if bch.Name == name && (i == -1 || i != bch_index) {
							Notification.ShowErrorf("Blockchain %s already exists", name)
							break
						}
					}

					rpc := v.GetInput("rpc")

					if len(rpc) == 0 {
						Notification.ShowError("RPC URL cannot be empty")
						break
					}

					chainid, err := strconv.Atoi(v.GetInput("chainid"))
					if err != nil || chainid <= 0 {
						Notification.ShowError("Invalid ChainId")
						break
					}

					for i, bch := range cmn.CurrentWallet.Blockchains {
						if bch.ChainID == chainid && (i == -1 || i != bch_index) {
							Notification.ShowErrorf("Chain id %d already used by %s", chainid, bch.Name)
							break
						}
					}

					explorer := v.GetInput("explorer")
					if len(explorer) == 0 {
						Notification.ShowError("Explorer URL cannot be empty")
						break
					}

					api_token := v.GetInput("api_token")

					currency := v.GetInput("currency")

					if len(currency) == 0 {
						Notification.ShowError("Currency cannot be empty")
						break
					}

					wtoken_address := v.GetInput("wtoken_address")
					if wtoken_address != "" {
						if !common.IsHexAddress(wtoken_address) {
							Notification.ShowError("Invalid Wrapped Token Address")
							break
						}
					}

					explorer_api_url := v.GetInput("explorer_api_url")
					explorer_api_type := v.GetInput("explorer_api_type")

					wta := common.HexToAddress(wtoken_address)

					if bch_index != -1 {
						cmn.CurrentWallet.Blockchains[bch_index].Name = name
						cmn.CurrentWallet.Blockchains[bch_index].Url = rpc
						cmn.CurrentWallet.Blockchains[bch_index].ChainID = chainid
						cmn.CurrentWallet.Blockchains[bch_index].ExplorerUrl = explorer
						cmn.CurrentWallet.Blockchains[bch_index].ExplorerAPIToken = api_token
						cmn.CurrentWallet.Blockchains[bch_index].ExplorerAPIUrl = explorer_api_url
						cmn.CurrentWallet.Blockchains[bch_index].ExplorerApiType = explorer_api_type
						cmn.CurrentWallet.Blockchains[bch_index].Currency = currency
						cmn.CurrentWallet.Blockchains[bch_index].WTokenAddress = wta
					} else {
						cmn.CurrentWallet.Blockchains = append(cmn.CurrentWallet.Blockchains, &cmn.Blockchain{
							Name:             name,
							Url:              rpc,
							ChainID:          chainid,
							ExplorerUrl:      explorer,
							ExplorerAPIToken: api_token,
							Currency:         currency,
							WTokenAddress:    wta,
						})
					}

					cmn.CurrentWallet.AuditNativeTokens()
					err = cmn.CurrentWallet.Save()
					if err != nil {
						Notification.ShowErrorf("Failed to save wallet: %s", err)
						return
					}

					Notification.Showf("Blockchain %s updated", name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: `
              Name: <input id:name size:32 value:"">
               RPC: <input id:rpc size:32 value:"">
           ChainId: <input id:chainid size:32 value:"">
          Currency: <input id:currency size:32 value:"">
Wrapped Token Addr: <input id:wtoken_address size:32 value:"">
<line text:Explorer>
               URL: <input id:explorer size:32 value:"">
           API URL: <input id:explorer_api_url size:32 value:""> 
          API Type: <select id:explorer_api_type size:16> 
         API Token: <input id:api_token size:32 value:"">

 <c><button text:Ok tip:"create wallet">  <button text:Cancel>`,
	}
}
