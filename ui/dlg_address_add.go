package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
)

func DlgAddressAdd(addr string, signer string, path string) *gocui.Popup {
	template := fmt.Sprintf(`
Address: %s	
   Name: <input id:name size:32 value:""> 
    Tag: <input id:tag size:32 value:"">
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
			dp = append(dp[:], make([]string, 7-len(dp))...) // pad with empty strings
			if len(dp) >= 5 {

				p3, _ := strconv.Atoi(strings.TrimSuffix(dp[3], "'"))
				p4, _ := strconv.Atoi(strings.TrimSuffix(dp[4], "'"))
				p5, _ := strconv.Atoi(strings.TrimSuffix(dp[5], "'"))

				if dp[5] == "" { // legasy
					name = signer + fmt.Sprintf("_L%d", p4)
				} else {
					name = signer + fmt.Sprintf("_%d", max(p3, p4, p5))
				}
			}

			v.SetInput("name", name)
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

					s := cmn.CurrentWallet.GetSigner(signer)
					if s == nil {
						Notification.ShowErrorf("Signer %s not found", signer)
						break
					}

					cmn.CurrentWallet.Addresses = append(cmn.CurrentWallet.Addresses, &cmn.Address{
						Name:    name,
						Address: a,
						Signer:  s.Name,
						Tag:     v.GetInput("tag"),
						Path:    path,
					})

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
