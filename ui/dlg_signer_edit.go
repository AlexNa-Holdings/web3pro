package ui

import (
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
)

func DlgSignerEdit(name string) *gocui.Popup {

	if wallet.CurrentWallet == nil {
		Notification.ShowError("No wallet open")
		return nil
	}

	signer_index := -1
	for i, s := range wallet.CurrentWallet.Signers {
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
		OnClose: func(v *gocui.View) {
			Gui.SetCurrentView("terminal.input")
		},
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				switch hs.Value {
				case "button Ok":

					new_name := v.GetInput("name")

					for i, signer := range wallet.CurrentWallet.Signers {
						if i != signer_index && signer.Name == new_name {
							Notification.ShowErrorf("Signer %s already exists", new_name)
							break
						}
					}

					old_name := wallet.CurrentWallet.Signers[signer_index].Name
					// update all the copies
					for _, signer := range wallet.CurrentWallet.Signers {
						if signer.CopyOf == old_name {
							signer.CopyOf = new_name
						}
					}

					wallet.CurrentWallet.Signers[signer_index].Name = new_name

					wallet.CurrentWallet.SortSigners()
					err := wallet.CurrentWallet.Save()
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
 Name: <i id:name size:32 value:"">
 Type: ` + wallet.CurrentWallet.Signers[signer_index].Type + `

 <c><b text:Ok tip:"create wallet">  <b text:Cancel>`,
	}
}
