package ui

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
)

func DlgAddressEdit(name string) *gocui.Popup {

	if wallet.CurrentWallet == nil {
		return nil
	}

	currect_a := wallet.CurrentWallet.GetAddressByName(name)
	if currect_a == nil {
		return nil
	}

	template := fmt.Sprintf(`
Address: %s	
   Name: <i id:name size:32 value:""> 
 Signer: %s
   Path: %s

<c><button text:Ok tip:"create wallet">  <button text:Cancel>`, currect_a.Address.String(), currect_a.Signer, currect_a.Path)

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
		},
		OnClose: func(v *gocui.View) {
			Gui.SetCurrentView("terminal.input")
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

					for _, a := range wallet.CurrentWallet.Addresses {
						if a != currect_a && a.Name == name {
							Notification.ShowError("Name already exists")
							break
						}
					}

					currect_a.Name = name

					err := wallet.CurrentWallet.Save()
					if err != nil {
						Notification.ShowErrorf("Error creating signer: %s", err)
						break
					}
					Notification.Showf("Signer %s created", name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: template,
	}
}
