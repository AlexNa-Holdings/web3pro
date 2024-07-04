package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/address"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
	"github.com/ethereum/go-ethereum/common"
)

func DlgAddressAdd(addr string, signer string, path string) *gocui.Popup {
	template := fmt.Sprintf(`
Address: %s	
   Name: <i id:name size:32 value:""> 
    Tag: <i id:tag size:32 value:"">
 Signer: %s
   Path: %s

<c><button text:Ok tip:"create wallet">  <button text:Cancel>`, addr, signer, path)

	return &gocui.Popup{
		Title: "Add address",
		// Subtitle: "Enter wallet name",
		OnOverHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				Bottom.Printf(hs.Tip)
			} else {
				Bottom.Printf("")
			}
		},
		OnOpen: func(v *gocui.View) {

			name := ""

			dp := strings.Split(path, "/")
			if len(dp) >= 5 {

				p3, _ := strconv.Atoi(strings.TrimSuffix(dp[3], "'"))
				p5, _ := strconv.Atoi(strings.TrimSuffix(dp[5], "'"))

				if p3 > 0 {
					name = signer + "_" + fmt.Sprintf("%d", p3)
				} else {
					name = signer + "_" + fmt.Sprintf("%d", p5)
				}
			}

			v.SetInput("name", name)
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

					a := common.HexToAddress(addr)

					if wallet.CurrentWallet.GetAddress(a.String()) != nil {
						Notification.ShowError("Address already exists")
						break
					}

					if wallet.CurrentWallet.GetAddressByName(name) != nil {
						Notification.ShowError("Name already exists")
						break
					}

					s := wallet.CurrentWallet.GetSigner(signer)
					if s == nil {
						Notification.ShowErrorf("Signer %s not found", signer)
						break
					}

					wallet.CurrentWallet.Addresses = append(wallet.CurrentWallet.Addresses, &address.Address{
						Name:    name,
						Address: a,
						Signer:  s.Name,
						Tag:     v.GetInput("tag"),
						Path:    path,
					})

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
