package ui

import "github.com/AlexNa-Holdings/web3pro/gocui"

func NewDlgWaletCreate() *gocui.Popup {
	return &gocui.Popup{
		Width:  40,
		Height: 10,
		Title:  "Create Wallet",
		// Subtitle: "Enter wallet name",
		Template: `
Password: <i masked:true len:10>
        <b text:Ok tip:"ha ha ha">  <b text:Cancel>`,
	}
}
