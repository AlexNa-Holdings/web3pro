package ui

import (
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
)

func DlgSignerEdit(name string) *gocui.Popup {

	if cmn.CurrentWallet == nil {
		Notification.ShowError("No wallet open")
		return nil
	}

	signer_index := -1
	for i, s := range cmn.CurrentWallet.Signers {
		if s.Name == name {
			signer_index = i
			break
		}
	}

	if signer_index == -1 {
		Notification.ShowErrorf("Signer %s not found", name)
		return nil
	}

	return &gocui.Popup{
		Title: "Edit Signer",
		OnOverHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				Bottom.Printf(hs.Tip)
			} else {
				Bottom.Printf("")
			}
		},
		OnOpen: func(v *gocui.View) {
			v.SetInput("name", name)
		},
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				switch hs.Value {
				case "button Ok":

					new_name := v.GetInput("name")

					for i, signer := range cmn.CurrentWallet.Signers {
						if i != signer_index && signer.Name == new_name {
							Notification.ShowErrorf("Signer %s already exists", new_name)
							break
						}
					}

					cmn.CurrentWallet.Signers[signer_index].Name = new_name

					err := cmn.CurrentWallet.Save()
					if err != nil {
						Notification.ShowErrorf("Error renaming signer: %s", err)
						break
					}

					Notification.Showf("Signer %s renamed", name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: `
 Name: <input id:name size:32 value:"">
 Type: ` + cmn.CurrentWallet.Signers[signer_index].Type + `
 <c>
 <button text:Ok tip:"create wallet">  <button text:Cancel>`,
	}
}
