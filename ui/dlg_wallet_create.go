package ui

import "github.com/AlexNa-Holdings/web3pro/gocui"

func DlgWaletCreate() *gocui.Popup {
	return &gocui.Popup{
		Title: "Create Wallet",
		// Subtitle: "Enter wallet name",
		OnOverHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				Bottom.Printf(hs.Tip)
			} else {
				Bottom.Printf("")
			}
		},
		Template: `
      Name: <i id:name size:24>
  Password: <i id:pass masked:true size:24>
  (Retype): <i id:retype masked:true size:24>

<c><b text:Ok tip:"ha ha ha">  <b text:Cancel>`,
	}
}
