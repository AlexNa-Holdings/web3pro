package ui

import "github.com/AlexNa-Holdings/web3pro/gocui"

func NewDlgWaletCreate() *gocui.Popup {
	return &gocui.Popup{
		Width:    50,
		Height:   10,
		Title:    "Create Wallet",
		Subtitle: "Enter wallet name",
	}
}
