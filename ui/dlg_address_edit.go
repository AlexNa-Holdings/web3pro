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

	template := fmt.Sprintf(`
Address: %s	
   Name: <i id:name size:32 value:""> 
    Tag: <i id:tag size:32 value:"">
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
			v.SetInput("tag", currect_a.Tag)
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
