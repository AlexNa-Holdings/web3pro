package ui

import (
	"encoding/hex"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/signer"
	"github.com/AlexNa-Holdings/web3pro/wallet"
	"github.com/tyler-smith/go-bip39"
)

func DlgSignerAdd(t string, sn string) *gocui.Popup {
	template := ""

	if t != "mnemonics" {
		template = `
   Name: <i id:name size:32 value:""> 
   Type: ` + t + `
     SN: <i id:sn size:32>
	 
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
 Mnemonic: <t id:sn width:32 height:8> 

 





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
			v.SetInput("name", "")
			v.SetInput("sn", sn)

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

					name := strings.TrimSpace(v.GetInput("name"))
					sn := strings.TrimSpace(v.GetInput("sn"))
					copyof := strings.TrimSpace(v.GetInput("copyof"))

					if len(name) == 0 {
						Notification.ShowError("Name cannot be empty")
						break
					}

					for _, signer := range wallet.CurrentWallet.Signers {
						if signer.Name == name {
							Notification.ShowErrorf("Signer %s already exists", name)
							return
						}
						for _, copy := range signer.Copies {
							if copy.Name == name {
								Notification.ShowErrorf("Signer %s already exists", name)
								return
							}
						}
					}

					if t == "mnemonics" {
						if len(sn) == 0 {
							Notification.ShowError("Mnemonic cannot be empty")
							break
						}

						m, err := bip39.EntropyFromMnemonic(sn)
						if err != nil {
							Notification.ShowError("Invalid mnemonics")
							break
						}

						sn = hex.EncodeToString(m[:])

						wallet.CurrentWallet.Signers = append(wallet.CurrentWallet.Signers, &signer.Signer{
							Name: name,
							Type: t,
							SN:   sn,
						})

					} else { // not mnemonics
						if len(sn) == 0 {
							Notification.ShowError("SN cannot be empty")
							break
						}

						for _, signer := range wallet.CurrentWallet.Signers {
							if t == signer.Type && signer.SN == sn {
								Notification.ShowErrorf("Signer %s has same SN", signer.Name)
								return
							}

							for _, copy := range signer.Copies {
								if copy.SN == sn {
									Notification.ShowErrorf("Signer %s has same SN", copy.Name)
									return
								}
							}
						}
						if copyof != "" {
							added := false
							for _, s := range wallet.CurrentWallet.Signers {
								if s.Name == copyof {
									s.Copies = append(s.Copies, signer.SignerCopy{
										Name: name,
										SN:   sn,
									})
									added = true
									break
								}
							}
							if !added {
								Notification.ShowErrorf("Signer %s not found", copyof)
								break
							}
						} else {
							wallet.CurrentWallet.Signers = append(wallet.CurrentWallet.Signers, &signer.Signer{
								Name: name,
								Type: t,
								SN:   sn,
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
