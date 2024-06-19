package ui

import (
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
)

func DlgBlockchainEdit(name string) *gocui.Popup {
	return &gocui.Popup{
		Title: "Edit Blockchain",
		OnOverHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				Bottom.Printf(hs.Tip)
			} else {
				Bottom.Printf("")
			}
		},
		OnClose: func(v *gocui.View) {
			Gui.SetCurrentView("terminal.input")
		},
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {

			if hs != nil {
				switch hs.Value {
				case "button Ok":
					name := v.GetInput("name")

					if len(name) == 0 {
						Notification.ShowError("Name cannot be empty")
						break
					}

					if wallet.Exists(name) {
						Notification.ShowErrorf("Wallet %s already exists", name)
						break
					}

					pass := v.GetInput("pass")
					retype := v.GetInput("retype")

					if pass != retype {
						Notification.ShowError("Passwords do not match")
						break
					}

					err := wallet.Create(name, pass)

					if err != nil {
						Notification.ShowErrorf("Error creating wallet: %s", err)
						break
					}

					Notification.Showf("Wallet %s created", name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: `
      Name: <i id:name size:32 value:"">
	  RPC:  <i id:rpc size:32 value:"">
      ChainId: <i id:chainid size:32 value:"">
      Explorer: <i id:explorer size:32 value:"">
      Currency: <i id:currency size:32 value:"">

 <c><b text:Ok tip:"create wallet">  <b text:Cancel>`,
	}
}
