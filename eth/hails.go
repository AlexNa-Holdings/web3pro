package eth

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
)

func HailToSend(b *cmn.Blockchain, t *cmn.Token, from *cmn.Address, to common.Address, amount *big.Int) {

	if cmn.CurrentWallet == nil {
		log.Error().Msg("No wallet")
		cmn.NotifyError("No wallet")
		return
	}

	w := cmn.CurrentWallet

	s := w.GetSigner(from.Signer)
	if s == nil {
		log.Error().Msg("Unknown signer: " + from.Signer)
		cmn.NotifyErrorf("Unknown signer: %v", from.Signer)
		return
	}

	to_name := ""
	to_addr := w.GetAddress(to.String())
	if to_addr != nil {
		to_name = to_addr.Name
	}

	tc := ""
	if !t.Native {
		tc = "\nToken Contract: " + cmn.AddressShortLinkTag(t.Address)
	}

	dollars := ""
	if t.Price > 0 {
		//		dollars = cmn.FormatDollarsNormal(t.Price*t.Float64(amount))
		dollars = cmn.FormatDollars(t.Price*t.Float64(amount), false)
	} else {
		dollars = "(unknown)"
	}

	var tx *types.Transaction
	var err error
	if t.Native {
		tx, err = BuildTxTransfer(b, s, from, to, amount)
	} else {
		tx, err = BuildTxERC20Transfer(b, t, s, from, to, amount)
	}

	if err != nil {
		log.Error().Err(err).Msg("Error building transaction")
		cmn.NotifyErrorf("Error building transaction: %v", err)
		return
	}

	total_gas := new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), tx.GasPrice())

	total_fee_s := "(unknown)"
	if t.Price > 0 {
		total_fee_dollars := t.Price * cmn.Float64(total_gas, 18)
		// total_fee_s = cmn.FormatDollarsNormal(total_fee_dollars)
		total_fee_s = cmn.FormatDollars(total_fee_dollars, false)
	}

	cmn.Hail(&cmn.HailRequest{
		Title: "Send Tokens",
		Template: `  Blockchain: ` + b.Name + `
       Token: ` + t.Symbol + tc + `
        From: ` + cmn.AddressShortLinkTag(from.Address) + " " + from.Name + `
          To: ` + cmn.AddressShortLinkTag(to) + " " + to_name + `
      Amount: ` + t.Value2Str(amount) + " " + t.Symbol + `
   Amount($): ` + dollars + ` 
      Signer: ` + s.Name + " (" + s.Type + ")" + `
<line text:Fee> 
   Gas Limit: ` + cmn.FormatUInt64(tx.Gas(), false, "") + ` 
   Gas Price: ` + cmn.FormatAmount(tx.GasPrice(), 18, false, "") + " " +
			b.Currency + ` <l text:` + gocui.ICON_EDIT + ` action:'button edit_gas_price' tip:"edit fee">
   Total Fee: ` + cmn.FormatAmount(total_gas, 18, false, "") + " " + b.Currency + `
Total Fee($): ` + total_fee_s + `
<c>
` +
			`<button text:Send id:ok bgcolor:g.HelpBgColor color:g.HelpFgColor tip:"send tokens">  ` +
			`<button text:Reject id:cancel bgcolor:g.ErrorFgColor tip:"reject transaction">`,
		OnOpen: func(hr *cmn.HailRequest, g *gocui.Gui, v *gocui.View) {
			v.SetInput("to", to.String())
			v.SetInput("amount", amount.String())
		},
		OnClose: func(hr *cmn.HailRequest) {
		},
		OnOk: func(hr *cmn.HailRequest) {
			if t.Native {
				err := Transfer(b, s, from, to, amount)
				if err != nil {
					log.Error().Err(err).Msg("Error sending native tokens")
					cmn.NotifyErrorf("Error sending native tokens: %v", err)
				}
			} else {
				err := ERC20Transfer(b, t, s, from, to, amount)
				if err != nil {
					log.Error().Err(err).Msg("Error sending tokens")
					cmn.NotifyErrorf("Error sending tokens: %v", err)
				}
			}
		},
		OnOverHotspot: func(hr *cmn.HailRequest, v *gocui.View, hs *gocui.Hotspot) {
			cmn.StandardOnOverHotspot(v, hs)
		},
		OnClickHotspot: func(hr *cmn.HailRequest, v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				switch hs.Value {
				case "button edit_gas_price":
					hr.TimerPaused = true

					low := new(big.Int).Div(new(big.Int).Mul(tx.GasPrice(), big.NewInt(9)), big.NewInt(10))
					market := tx.GasPrice()
					high := new(big.Int).Div(new(big.Int).Mul(tx.GasPrice(), big.NewInt(11)), big.NewInt(10))

					var p_low, p_market, p_high string
					if t.Price > 0 {
						// p_low = cmn.FormatDollarsNormal(t.Price * cmn.Float64(high, 18) * float64(tx.Gas()))
						// p_market = cmn.FormatDollarsNormal(t.Price * cmn.Float64(market, 18) * float64(tx.Gas()))
						// p_high = cmn.FormatDollarsNormal(t.Price * cmn.Float64(high, 18) * float64(tx.Gas()))
						p_low = cmn.FormatDollars(t.Price*cmn.Float64(high, 18)*float64(tx.Gas()), true)
						p_market = cmn.FormatDollars(t.Price*cmn.Float64(market, 18)*float64(tx.Gas()), true)
						p_high = cmn.FormatDollars(t.Price*cmn.Float64(high, 18)*float64(tx.Gas()), true)
					}

					v.GetGui().ShowPopup(&gocui.Popup{
						Title: "Edit Gas Price",
						Template: `<c><w>
 <button text:' Low  '> ` + cmn.FormatAmount(low, 18, true, "") + p_low + `
 <button text:'Market'> ` + cmn.FormatAmount(market, 18, true, "") + p_market + `
 <button text:' High '> ` + cmn.FormatAmount(high, 18, true, "") + p_high + `

 <line text:Advanced>
Fee price: <input id:gas_price size:14 value:"` + cmn.FormatAmount(market, 18, false, "") + `"> 
 Total($): <input id:gas_price_dollars size:14 value:"` +
							fmt.Sprintf("%f", t.Price*cmn.Float64(market, 18)*float64(tx.Gas())) + `"> 

 <button text:'Use custom price'>
<line>

<button text:Cancel>`,
						OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
							if hs != nil {
								switch hs.Value {
								case "button OK":
									v.GetGui().HidePopup()
									hr.Close()
								case "button Cancel":
									v.GetGui().HidePopup()
								}
							}
						},
						OnClose: func(v *gocui.View) {
							hr.ResetTimer()
							hr.TimerPaused = false
						},
						OnOpen: func(v *gocui.View) {
							v.SetFocus(1) // market
						},
					})
				}
			}

			cmn.StandardOnClickHotspot(v, hs)
		},
	})

}
