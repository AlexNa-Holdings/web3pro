package ui

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
)

func DlgAddressEdit(name string) *gocui.Popup {

	if cmn.CurrentWallet == nil {
		return nil
	}

	currect_a := cmn.CurrentWallet.GetAddressByName(name)
	if currect_a == nil {
		return nil
	}

	signerInfo := currect_a.Signer
	pathInfo := currect_a.Path
	if signerInfo == "" {
		signerInfo = "(watch-only)"
		pathInfo = ""
	}

	template := fmt.Sprintf(`
Address: %s
   Name: <input id:name size:32 value:"">
    Tag: <input id:tag size:32 value:"">
 Signer: %s
   Path: %s

<c><button text:Ok tip:"save changes">  <button text:Cancel>`, currect_a.Address.String(), signerInfo, pathInfo)

	return &gocui.Popup{
		Title: "Edit address",
		OnOverHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				Bottom.Printf(hs.Tip)
			} else {
				Bottom.Printf("")
			}
		},
		OnOpen: func(v *gocui.View) {
			v.SetInput("name", currect_a.Name)
			v.SetInput("tag", currect_a.Tag)
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

					for _, a := range cmn.CurrentWallet.Addresses {
						if a != currect_a && a.Name == name {
							Notification.ShowError("Name already exists")
							break
						}
					}

					currect_a.Name = name
					currect_a.Tag = v.GetInput("tag")

					err := cmn.CurrentWallet.Save()
					if err != nil {
						Notification.ShowErrorf("Error saving address: %s", err)
						break
					}
					Notification.Showf("Address %s updated", name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: template,
	}
}
