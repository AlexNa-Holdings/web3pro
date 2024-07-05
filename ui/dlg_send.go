package ui

import (
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
)

func DlgSend(b *cmn.Blockchain, t *cmn.Token, from, to string, amount string) *gocui.Popup {
	tc := t.Address.String()
	if t.Native {
		tc = "Native"
	}

	template := `
     Blockchain: ` + b.Name + `
          Token: ` + t.Symbol + `
 Token Contract: ` + tc + ` 
           From: ` + from + `
             To: <i id:to size:32> 
         Amount: <i id:amount size:32> 
 <c>
 <button text:Ok tip:"create wallet">  <button text:Cancel>`

	return &gocui.Popup{
		Title: "Send Tokens",
		OnOpen: func(v *gocui.View) {
			v.SetInput("to", to)
			v.SetInput("amount", amount)
		},
		OnClose: func(v *gocui.View) {
			Gui.SetCurrentView("terminal.input")
		},
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {

			if hs != nil {
				switch hs.Value {
				case "button Ok":
					// pass := v.GetInput("pass")

					// err := wallet.Open(name, pass)

					// if err != nil {
					// 	Notification.ShowErrorf("Error opening wallet: %s", err)
					// 	v.SetInput("pass", "")
					// 	v.SetFocus(0)
					// 	break
					// }

					// Notification.Showf("Wallet %s open", name)
					// Terminal.SetCommandPrefix(name)
					// Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: template,
	}
}
