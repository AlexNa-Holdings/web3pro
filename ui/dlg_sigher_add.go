package ui

import (
	"encoding/hex"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/signer"
	"github.com/AlexNa-Holdings/web3pro/wallet"
	"github.com/tyler-smith/go-bip39"
)

func DlgSignerAdd(t string, name string) *gocui.Popup {
	template := ""

	if t != "mnemonics" {
		template = `
   Name: ` + name + `
   Type: ` + t + `
	 
Copy of: <select id:copyof size:32 value:""> <c>
NOTE: You are responsible to assure that the 
device marked as copy is the same as the one
you are adding. If you are not sure, please
cancel and verify the device.

<button text:Ok tip:"create wallet">  <button text:Cancel>`
	} else {
		template = `
     Name: <i id:name size:32 value:""> 
     Type: ` + t + `
 Mnemonic: <t id:mnemonics width:32 height:8> 

 





<c>
<button text:Ok tip:"create wallet">  <button text:Cancel>`
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
			v.SetInput("name", name)

			names := []string{""}
			for _, signer := range wallet.CurrentWallet.Signers {
				if signer.Type == t {
					names = append(names, signer.Name)
				}
			}

			v.SetList("copyof", names)
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

					copyof := strings.TrimSpace(v.GetInput("copyof"))

					if t == "mnemonics" {
						name := strings.TrimSpace(v.GetInput("name"))
						if len(name) == 0 {
							Notification.ShowError("Name cannot be empty")
							break
						}

						if wallet.CurrentWallet.GetSigner(name) != nil {
							Notification.ShowErrorf("Signer %s already exists", name)
							break
						}

						mm := strings.TrimSpace(v.GetInput("mnemonics"))
						if len(mm) == 0 {
							Notification.ShowError("Mnemonic cannot be empty")
							break
						}

						m, err := bip39.EntropyFromMnemonic(mm)
						if err != nil {
							Notification.ShowError("Invalid mnemonics")
							break
						}

						wallet.CurrentWallet.Signers = append(wallet.CurrentWallet.Signers, &signer.Signer{
							Name: name,
							Type: t,
							SN:   hex.EncodeToString(m[:]),
						})

					} else { // not mnemonics

						if wallet.CurrentWallet.GetSigner(name) != nil {
							Notification.ShowErrorf("Signer %s already exists", name)
							break
						}

						if copyof != "" {
							ms := wallet.CurrentWallet.GetSigner(copyof)
							if ms != nil {
								ms.Copies = append(ms.Copies, name)
							} else {
								Notification.ShowErrorf("Signer %s not found", copyof)
								break
							}
						} else {
							wallet.CurrentWallet.Signers = append(wallet.CurrentWallet.Signers, &signer.Signer{
								Name: name,
								Type: t,
							})
						}
					}

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
