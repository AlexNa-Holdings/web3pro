package ui

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

func DlgSend(b *cmn.Blockchain, t *cmn.Token, from *cmn.Address, to string, amount string) *gocui.Popup {

	tc := cmn.AddressShortLinkTag(t.Address)
	if t.Native {
		tc = "Native"
	}

	template := `
     Blockchain: ` + b.Name + `
          Token: <b>` + t.Symbol + `</b>
 Token Contract: ` + tc + ` 
           From: ` + from.Name + `
             To: <input id:to size:43> 
         Amount: <input id:amount size:24> 
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

					if !common.IsHexAddress(to) {
						bus.Send("ui", "notify-error", fmt.Sprintf("Invalid address: %s", to))
						return
					}

					val, err := t.Str2Wei(amount)
					if err != nil || val.Cmp(big.NewInt(0)) <= 0 {
						log.Error().Err(err).Msgf("Str2Value(%s) err: %v", amount, err)
						bus.Send("ui", "notify-error", fmt.Sprintf("Invalid amount: %s", amount))
						return
					}

					bus.Send("eth", "send", &bus.B_EthSend{
						Blockchain: b.Name,
						Token:      t.Symbol,
						From:       from.Address,
						To:         common.HexToAddress(to),
						Amount:     val,
					})

					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: template,
	}
}
