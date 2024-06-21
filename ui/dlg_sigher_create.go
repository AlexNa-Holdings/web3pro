package ui

import (
	"encoding/hex"

	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/signer"
	"github.com/AlexNa-Holdings/web3pro/wallet"
	"github.com/tyler-smith/go-bip39"
)

func DlgSignerCreate(t string, sn string) *gocui.Popup {
	template := ""

	if t != "mnemonic" {
		template = `
 Name: <i id:name size:32 value:"">
 Type: ` + t + `
   SN: <i id:sn size:32>

<c><b text:Ok tip:"create wallet">  <b text:Cancel>`
	} else {
		template = `
     Name: <i id:name size:32 value:"">
     Type: ` + t + `
 Mnemonic: <t id:sn width:32 height:8>






 
<c><b text:Ok tip:"create wallet">  <b text:Cancel>`
	}

	return &gocui.Popup{
		Title: "Create Signer",
		// Subtitle: "Enter wallet name",
		OnOverHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				Bottom.Printf(hs.Tip)
			} else {
				Bottom.Printf("")
			}
		},
		OnOpen: func(v *gocui.View) {
			v.SetInput("name", "")
			v.SetInput("sn", sn)
		},
		OnClose: func(v *gocui.View) {
			Gui.SetCurrentView("terminal.input")
		},
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {

			if hs != nil {
				switch hs.Value {
				case "button Ok":
					if wallet.CurrentWallet == nil {
						Notification.ShowError("No wallet open")
						break
					}

					name := v.GetInput("name")
					sn := v.GetInput("sn")

					if len(name) == 0 {
						Notification.ShowError("Name cannot be empty")
						break
					}

					for _, signer := range wallet.CurrentWallet.Signers {
						if signer.Name == name {
							Notification.ShowErrorf("Signer %s already exists", name)
							break
						}
					}

					if t == "mnemonic" {
						if len(sn) == 0 {
							Notification.ShowError("Mnemonic cannot be empty")
							break
						}

						m, err := bip39.EntropyFromMnemonic(sn)
						if err != nil {
							Notification.ShowError("Invalid mnemonic")
							break
						}

						sn = hex.EncodeToString(m[:])

					} else {
						if len(sn) == 0 {
							Notification.ShowError("SN cannot be empty")
							break
						}
					}

					wallet.CurrentWallet.Signers = append(wallet.CurrentWallet.Signers, signer.Signer{Name: name, Type: t, SN: sn})
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
