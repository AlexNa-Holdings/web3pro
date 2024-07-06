package ui

import (
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/eth"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

func DlgSend(b *cmn.Blockchain, t *cmn.Token, from, to string, amount string) *gocui.Popup {
	tc := cmn.AddressShortLinkTag(t.Address)
	if t.Native {
		tc = "Native"
	}

	template := `
     Blockchain: ` + b.Name + `
          Token: <b>` + t.Symbol + `/<b>
 Token Contract: ` + tc + ` 
           From: ` + from + `
             To: <i id:to size:43> 
         Amount: <i id:amount size:24> 
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
					to := v.GetInput("to")
					amount := v.GetInput("amount")

					if !common.IsHexAddress(from) {
						Notification.ShowErrorf("Invalid address: %s", from)
						return
					}

					if !common.IsHexAddress(to) {
						Notification.ShowErrorf("Invalid address: %s", to)
						return
					}

					val, err := t.Str2Value(amount)
					if err != nil {
						log.Error().Err(err).Msgf("Str2Value(%s) err: %v", amount, err)
						Notification.ShowErrorf("Invalid amount: %s", amount)
						return
					}

					eth.HailToSend(b, t, common.HexToAddress(from), common.HexToAddress(to), val)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: template,
	}
}
