package ui

import (
	"fmt"
	"strconv"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
)

// chain_id = 0  means add new custom blockchain
func DlgBlockchain(chain_id int) *gocui.Popup {

	w := cmn.CurrentWallet
	if w == nil {
		Notification.ShowError("No wallet open")
		return nil
	}

	chain_line := ""
	if chain_id == 0 {
		chain_line = `           ChainId: <input id:chainid size:16 value:""> `
	} else {
		chain_line = fmt.Sprintf("           ChainId: %d ", chain_id)
	}

	b := w.GetBlockchain(chain_id)
	if chain_id != 0 && b == nil {
		Notification.ShowErrorf("Blockchain id %d not found", chain_id)
		return nil
	}

	title := "Add Blockchain"
	if chain_id != 0 {
		title = "Edit Blockchain"
	}

	return &gocui.Popup{
		Title:         title,
		OnOverHotspot: cmn.StandardOnOverHotspot,
		OnOpen: func(v *gocui.View) {
			v.SetSelectList("explorer_api_type", cmn.EXPLORER_API_TYPES)
			if b != nil {
				v.SetInput("name", b.Name)
				v.SetInput("rpc", b.Url)
				v.SetInput("chainid", strconv.Itoa(chain_id))
				v.SetInput("explorer", b.ExplorerUrl)
				v.SetInput("api_token", b.ExplorerAPIToken)
				v.SetInput("currency", b.Currency)
				if b.Multicall != (common.Address{}) {
					v.SetInput("multicall", b.Multicall.String())
				}
				if b.WTokenAddress != (common.Address{}) {
					v.SetInput("wtoken_address", b.WTokenAddress.String())
				}
				v.SetInput("explorer_api_url", b.ExplorerAPIUrl)
				v.SetInput("explorer_api_type", b.ExplorerApiType)
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

					bn := w.GetBlockchainByName(name)
					if bn != nil && (chain_id == 0 || bn.ChainId != chain_id) {
						Notification.ShowErrorf("Blockchain %s already exists", name)
						break
					}

					rpc := v.GetInput("rpc")

					if len(rpc) == 0 {
						Notification.ShowError("RPC URL cannot be empty")
						break
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

					multicall := v.GetInput("multicall")

					wta := common.HexToAddress(wtoken_address)

					var err error
					if chain_id != 0 {
						err = w.EditBlockchain(&cmn.Blockchain{
							Name:             name,
							Url:              rpc,
							ChainId:          chain_id,
							ExplorerUrl:      explorer,
							ExplorerAPIToken: api_token,
							ExplorerAPIUrl:   explorer_api_url,
							ExplorerApiType:  explorer_api_type,
							Currency:         currency,
							WTokenAddress:    wta,
							Multicall:        common.HexToAddress(multicall),
						})
					} else {
						var chainid int
						chainid, err = strconv.Atoi(v.GetInput("chainid"))
						if err != nil || chainid <= 0 {
							Notification.ShowError("Invalid ChainId")
							break
						}

						if w.GetBlockchain(chainid) != nil {
							Notification.ShowErrorf("ChainId %d already exists", chainid)
							break
						}

						err = w.AddBlockchain(&cmn.Blockchain{
							Name:             name,
							Url:              rpc,
							ChainId:          chainid,
							ExplorerUrl:      explorer,
							ExplorerAPIToken: api_token,
							ExplorerAPIUrl:   explorer_api_url,
							ExplorerApiType:  explorer_api_type,
							Currency:         currency,
							WTokenAddress:    wta,
							Multicall:        common.HexToAddress(multicall),
						})
					}

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
               RPC: <input id:rpc size:43 value:""> 
` + chain_line + `
          Currency: <input id:currency size:16 value:""> 
Wrapped Token Addr: <input id:wtoken_address size:43 value:""> 
Multicall Contract: <input id:multicall size:43 value:"">
<line text:Explorer> 
               URL: <input id:explorer size:43 value:"">
           API URL: <input id:explorer_api_url size:43 value:""> 
          API Type: <select id:explorer_api_type size:16> 
         API Token: <input id:api_token size:43 value:""> 

 <c><button text:Ok tip:"create wallet">  <button text:Cancel>`,
	}
}
