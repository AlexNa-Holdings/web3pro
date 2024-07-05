package ui

import (
	"strconv"

	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
)

func DlgBlockchainEdit(name string) *gocui.Popup {

	if wallet.CurrentWallet == nil {
		Notification.ShowError("No wallet open")
		return nil
	}

	bch_index := -1
	for i, bch := range wallet.CurrentWallet.Blockchains {
		if bch.Name == name {
			bch_index = i
			break
		}
	}

	if bch_index == -1 {
		Notification.ShowErrorf("Blockchain %s not found", name)
		return nil
	}

	return &gocui.Popup{
		Title: "Edit Blockchain",
		OnOverHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				Bottom.Printf(hs.Tip)
			} else {
				Bottom.Printf("")
			}
		},
		OnOpen: func(v *gocui.View) {
			bch := wallet.CurrentWallet.Blockchains[bch_index]
			v.SetInput("name", bch.Name)
			v.SetInput("rpc", bch.Url)
			v.SetInput("chainid", strconv.Itoa(int(bch.ChainId)))
			v.SetInput("explorer", bch.ExplorerUrl)
			v.SetInput("currency", bch.Currency)
		},
		OnClose: func(v *gocui.View) {
			Gui.SetCurrentView("terminal.input")
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

					for i, bch := range wallet.CurrentWallet.Blockchains {
						if bch.Name == name && i != bch_index {
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

					explorer := v.GetInput("explorer")

					if len(explorer) == 0 {
						Notification.ShowError("Explorer URL cannot be empty")
						break
					}

					currency := v.GetInput("currency")

					if len(currency) == 0 {
						Notification.ShowError("Currency cannot be empty")
						break
					}

					wallet.CurrentWallet.Blockchains[bch_index].Name = name
					wallet.CurrentWallet.Blockchains[bch_index].Url = rpc
					wallet.CurrentWallet.Blockchains[bch_index].ChainId = uint(chainid)
					wallet.CurrentWallet.Blockchains[bch_index].ExplorerUrl = explorer
					wallet.CurrentWallet.Blockchains[bch_index].Currency = currency

					err = wallet.CurrentWallet.Save()
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
          Name: <i id:name size:32 value:"">
           RPC: <i id:rpc size:32 value:"">
       ChainId: <i id:chainid size:32 value:"">
      Explorer: <i id:explorer size:32 value:"">
      Currency: <i id:currency size:32 value:"">

 <c><button text:Ok tip:"create wallet">  <button text:Cancel>`,
	}
}
