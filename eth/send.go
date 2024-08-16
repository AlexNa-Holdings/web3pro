package eth

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
)

func send(msg *bus.Message) error {
	req, ok := msg.Data.(*bus.B_EthSend)

	if !ok {
		return bus.ErrInvalidMessageData
	}

	w := cmn.CurrentWallet
	if w == nil {
		return errors.New("no wallet")
	}

	b := w.GetBlockchain(req.Blockchain)
	if b == nil {
		return fmt.Errorf("blockchain not found: %v", req.Blockchain)
	}

	t := w.GetToken(req.Blockchain, req.Token)
	if t == nil {
		return fmt.Errorf("token not found: %v", req.Token)
	}

	from := w.GetAddress(req.From.String())
	if from == nil {
		return fmt.Errorf("address from not found: %v", req.From)
	}

	template, err := BuildHailToSendTemplate(b, t, from, req.To, req.Amount, nil)
	if err != nil {
		log.Error().Err(err).Msg("Error building hail template")
		cmn.NotifyErrorf("Error: %v", err)
		return err
	}

	nt, _ := w.GetNativeToken(b)
	s := w.GetSigner(from.Signer)
	amount := req.Amount
	to := req.To

	var tx *types.Transaction
	if t.Native {
		tx, _ = BuildTxTransfer(b, s, from, to, amount)
	} else {
		tx, _ = BuildTxERC20Transfer(b, t, s, from, to, amount)
	}

	bus.Fetch("ui", "hail", &bus.B_Hail{
		Title:    "Send Tokens",
		Template: template,
		OnOpen: func(m *bus.Message, g *gocui.Gui, v *gocui.View) {
			v.SetInput("to", to.String())
			v.SetInput("amount", amount.String())
		},
		OnOk: func(m *bus.Message) {
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
		OnOverHotspot: func(m *bus.Message, v *gocui.View, hs *gocui.Hotspot) {
			cmn.StandardOnOverHotspot(v, hs)
		},
		OnClickHotspot: func(m *bus.Message, v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				switch hs.Value {
				case "button edit_gas_price":
					res := bus.Fetch("timer", "pause", m.TimerID)
					if res.Error != nil {
						log.Error().Err(res.Error).Msg("Error pausing timer")
						return
					}
					editFee(m, v, tx, t, nt, func(newGasPrice *big.Int) {
						template, err := BuildHailToSendTemplate(b, t, from, to, amount, newGasPrice)
						if err != nil {
							log.Error().Err(err).Msg("Error building hail template")
							cmn.NotifyErrorf("Error: %v", err)
							return
						}

						v.GetGui().UpdateAsync(func(*gocui.Gui) error {
							hail, ok := m.Data.(*bus.B_Hail)
							if ok {
								hail.Template = template
								v.RenderTemplate(template)
							}
							return nil
						})

						bus.Fetch("timer", "resume", m.TimerID)
					})
				default:
					cmn.StandardOnClickHotspot(v, hs)
				}
			}
		},
	})

	return nil
}

func editFee(_ *bus.Message, v *gocui.View, tx *types.Transaction, t *cmn.Token, nt *cmn.Token, on_close func(*big.Int)) {
	low := new(big.Int).Div(new(big.Int).Mul(tx.GasPrice(), big.NewInt(9)), big.NewInt(10))
	market := tx.GasPrice()
	high := new(big.Int).Div(new(big.Int).Mul(tx.GasPrice(), big.NewInt(11)), big.NewInt(10))

	var p_low, p_market, p_high, cp string
	if t.Price > 0 {
		p_low = cmn.FmtFloat64D(nt.Price*cmn.Float64(high, 18)*float64(tx.Gas()), true)
		p_market = cmn.FmtFloat64D(nt.Price*cmn.Float64(market, 18)*float64(tx.Gas()), true)
		p_high = cmn.FmtFloat64D(nt.Price*cmn.Float64(high, 18)*float64(tx.Gas()), true)

		cp = `
Total($): <input id:gas_price_dollars size:14 value:"` +
			cmn.FmtFloat64(nt.Price*cmn.Float64(market, 18)*float64(tx.Gas()), false) + `">`
	}

	newGasPrice := tx.GasPrice()

	v.GetGui().ShowPopup(&gocui.Popup{
		Title: "Edit Gas Price",
		Template: `<c><w>
<button text:' Low  '    id:Low> ` + cmn.FmtAmount(low, 18, true) + p_low + `
<button text:'Market' id:Market> ` + cmn.FmtAmount(market, 18, true) + p_market + `
<button text:' High '   id:High> ` + cmn.FmtAmount(high, 18, true) + p_high + `

<line text:Advanced></c>

Gas price: <input id:gas_price size:14 value:"` + cmn.FmtAmount(market, 18, false) + `"> ` + nt.Name + cp + ` 
<c>
<button text:'Use custom price' id:Custom>
<line>

<button text:Cancel>
`,
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				switch hs.Value {
				case "button Low":
					newGasPrice = low
					v.GetGui().HidePopup()
				case "button Market":
					newGasPrice = market
					v.GetGui().HidePopup()
				case "button High":
					newGasPrice = high
					v.GetGui().HidePopup()
				case "button Custom":
					gp := v.GetInput("gas_price")

					val, err := cmn.Str2Wei(gp, 18)
					if err != nil || val.Cmp(big.NewInt(0)) <= 0 {
						cmn.NotifyErrorf("Invalid gas price: %v", err)
						return
					}
					newGasPrice = val
					v.GetGui().HidePopup()

				case "button Cancel":
					v.GetGui().HidePopup()
				}
			}
		},
		OnChange: func(p *gocui.Popup, pc *gocui.PopoupControl) {
			if nt.Price <= 0 {
				return
			}
			switch pc.ID {
			case "gas_price":
				gp := p.GetInput("gas_price")

				val, err := cmn.Str2Wei(gp, 18)
				if err != nil || val.Cmp(big.NewInt(0)) <= 0 {
					cmn.NotifyErrorf("Invalid gas price: %v", err)
					return
				}

				total_gas := new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), val)
				total_fee_dollars := nt.Price * cmn.Float64(total_gas, 18)

				p.SetInput("gas_price_dollars", cmn.FmtFloat64(total_fee_dollars, false))
			case "gas_price_dollars":
				gpd := p.GetInput("gas_price_dollars")

				val, err := cmn.ParseXF(gpd)
				if err != nil || val.IsZero() {
					cmn.NotifyErrorf("Invalid dollar price: %v", err)
					return
				}

				val = val.Div(cmn.NewXF_Float64(nt.Price))
				val = val.Div(cmn.NewXF_UInt64(tx.Gas()))
				p.SetInput("gas_price", val.Format(false, ""))
			}
		},
		OnClose: func(v *gocui.View) {
			on_close(newGasPrice)
		},
		OnOpen: func(v *gocui.View) {
			v.SetFocus(1) // market
		},
	})
}

func BuildHailToSendTemplate(b *cmn.Blockchain, t *cmn.Token,
	from *cmn.Address, to common.Address, amount *big.Int, suggested_gas_price *big.Int) (string, error) {
	if cmn.CurrentWallet == nil {
		return "", errors.New("no wallet")
	}

	w := cmn.CurrentWallet

	nt, err := w.GetNativeToken(b)
	if err != nil {
		return "", err
	}

	s := w.GetSigner(from.Signer)
	if s == nil {
		return "", errors.New("signer not found")
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
		dollars = cmn.TagShortDollarLink(t.Price * t.Float64(amount))
	} else {
		dollars = "(unknown)"
	}

	var tx *types.Transaction
	if t.Native {
		tx, err = BuildTxTransfer(b, s, from, to, amount)
	} else {
		tx, err = BuildTxERC20Transfer(b, t, s, from, to, amount)
	}

	if err != nil {
		log.Error().Err(err).Msg("Error building transaction")
		return "", err
	}

	gas_price := tx.GasPrice()
	gp_change := ""
	if suggested_gas_price != nil && suggested_gas_price.Cmp(gas_price) != 0 {
		if suggested_gas_price.Cmp(gas_price) < 0 {
			percents := new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(gas_price, suggested_gas_price), big.NewInt(100)), gas_price)
			f := float64(percents.Int64())
			gp_change = fmt.Sprintf(` <color fg:green>↓%2.2f%%`, f)
			if f > 10 {
				gp_change += "\n<c><blink>TOO LOW</blink></c>"
			}
			gp_change += `</color>`
		} else {
			percents := new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(suggested_gas_price, gas_price), big.NewInt(100)), gas_price)
			f := float64(percents.Int64())
			gp_change = fmt.Sprintf(` <color fg:red>↑%2.2f%%`, f)
			if f > 10 {
				gp_change += "\n<c><blink>TOO HIGH</blink></c>"
			}
			gp_change += `</color>`
		}

		log.Debug().Msgf("go_change: %v", gp_change)

		gas_price = suggested_gas_price
	}
	total_gas := new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), gas_price)

	total_fee_s := "(unknown)"
	if nt.Price > 0 {
		total_fee_dollars := nt.Price * cmn.Float64(total_gas, 18)
		// total_fee_s = cmn.FormatDollarsNormal(total_fee_dollars)
		total_fee_s = cmn.TagShortDollarLink(total_fee_dollars)
	}

	return `  Blockchain: ` + b.Name + `
       Token: ` + t.Symbol + tc + `
        From: ` + cmn.AddressShortLinkTag(from.Address) + " " + from.Name + `
          To: ` + cmn.AddressShortLinkTag(to) + " " + to_name + `
      Amount: ` + t.Value2Str(amount) + " " + t.Symbol + `
   Amount($): ` + dollars + ` 
      Signer: ` + s.Name + " (" + s.Type + ")" + `
<line text:Fee> 
   Gas Limit: ` + cmn.FormatUInt64(tx.Gas(), false) + ` 
   Gas Price: ` + cmn.TagValueSymbolLink(gas_price, nt) + " " +
		` <l text:` + gocui.ICON_EDIT + ` action:'button edit_gas_price' tip:"edit fee">` + gp_change + `
   Total Fee: ` + cmn.TagValueSymbolLink(total_gas, nt) + `
Total Fee($): ` + total_fee_s + `
<c>
` +
		`<button text:Send id:ok bgcolor:g.HelpBgColor color:g.HelpFgColor tip:"send tokens">  ` +
		`<button text:Reject id:cancel bgcolor:g.ErrorFgColor tip:"reject transaction">`, nil
}
