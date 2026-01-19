package ui

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
)

func DlgAddressAddWatch(addr string) *gocui.Popup {
	template := fmt.Sprintf(`
Address: %s
   Name: <input id:name size:32 value:"">
    Tag: <input id:tag size:32 value:"">
   Type: (watch-only)

<c><button text:Ok tip:"add watch-only address">  <button text:Cancel>`, addr)

	return &gocui.Popup{
		Title: "Add Watch Address",
		OnOverHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				Bottom.Printf(hs.Tip)
			} else {
				Bottom.Printf("")
			}
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

					a := common.HexToAddress(addr)

					if cmn.CurrentWallet.GetAddress(a.String()) != nil {
						Notification.ShowError("Address already exists")
						break
					}

					if cmn.CurrentWallet.GetAddressByName(name) != nil {
						Notification.ShowError("Name already exists")
						break
					}

					cmn.CurrentWallet.Addresses = append(cmn.CurrentWallet.Addresses, &cmn.Address{
						Name:    name,
						Address: a,
						Signer:  "", // Empty signer = watch-only
						Tag:     v.GetInput("tag"),
						Path:    "",
					})

					err := cmn.CurrentWallet.Save()
					if err != nil {
						Notification.ShowErrorf("Error adding address: %s", err)
						break
					}
					Notification.Showf("Watch address %s added", name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: template,
	}
}
