package eth

import (
	"context"
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

func sendTx(msg *bus.Message) error {
	req, ok := msg.Data.(*bus.B_EthSendTx)
	if !ok {
		return fmt.Errorf("invalid tx: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return errors.New("no wallet")
	}

	b := w.GetBlockchain(req.Blockchain)
	if b == nil {
		return fmt.Errorf("blockchain not found: %v", req.Blockchain)
	}

	from := w.GetAddress(req.From.String())
	if from == nil {
		return fmt.Errorf("address from not found: %v", req.From)
	}

	template, err := BuildHailToSendTxTemplate(b, from, req.To, req.Amount, req.Data, nil)
	if err != nil {
		log.Error().Err(err).Msg("Error building send-tx hail template")
		bus.Send("ui", "notify-error", fmt.Sprintf("Error: %v", err))
		return err
	}

	nt, _ := w.GetNativeToken(b)

	tx, err := BuildTx(b, w.GetSigner(from.Signer), from, req.To, req.Amount, req.Data)
	if err != nil {
		log.Error().Err(err).Msg("Error building transaction")
		bus.Send("ui", "notify-error", fmt.Sprintf("Error: %v", err))
		return err
	}

	msg.Fetch("ui", "hail", &bus.B_Hail{
		Title:    "Send Tx",
		Template: template,
		OnOk: func(m *bus.Message) {
			//TODO
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
					editFee(m, v, tx, nt, func(newGasPrice *big.Int) {
						template, err := BuildHailToSendTxTemplate(b, from, req.To, req.Amount, req.Data, newGasPrice)
						if err != nil {
							log.Error().Err(err).Msg("Error building hail template")
							bus.Send("ui", "notify-error", fmt.Sprintf("Error: %v", err))
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

	/////////////////////////////////////////////////////////
	// Sign the transaction
	// sign_res := bus.Fetch("signer", "sign-tx", &bus.B_SignerSignTx{
	// 	Type:      signer.Type,
	// 	Name:      signer.Name,
	// 	MasterKey: signer.MasterKey,
	// 	Chain:     b.Name,
	// 	Tx:        tx,
	// 	From:      from.Address,
	// 	Path:      from.Path,
	// })

	// if sign_res.Error != nil {
	// 	return fmt.Errorf("error signing transaction: %v", sign_res.Error)
	// }

	// signedTx, ok := sign_res.Data.(*types.Transaction)
	// if !ok {
	// 	log.Error().Msgf("Transfer: Cannot convert to transaction. Data:(%v)", sign_res.Data)
	// 	return errors.New("cannot convert to transaction")
	// }

	// send_res := bus.Fetch("eth", "send-tx", signedTx)
	// if res.Error != nil {
	// 	log.Error().Err(send_res.Error).Msg("Transfer: Cannot send tx")
	// 	return send_res.Error
	// }

	// c, ok := cons[int(signedTx.ChainId().Int64())]
	// if !ok {
	// 	return fmt.Errorf("client not found for chainId: %v", signedTx.ChainId())
	// }

	// // Send the transaction
	// err := c.SendTransaction(context.Background(), signedTx)
	// if err != nil {
	// 	log.Error().Err(err).Msgf("Transfer: Cannot send transaction")
	// 	return err
	// }

	//bus.Send("ui", "notify", fmt.Sprintf("Transaction sent: %s", signedTx.Hash().Hex()))

	return nil
}

func BuildTx(b *cmn.Blockchain, s *cmn.Signer, from *cmn.Address, to common.Address,
	amount *big.Int, data []byte) (*types.Transaction, error) {

	if from.Signer != s.Name {
		log.Error().Msgf("BuildTxTransfer: Signer mismatch. Token:(%s) Blockchain:(%s)", from.Signer, s.Name)
		return nil, errors.New("signer mismatch")
	}

	client, err := getEthClient(b)
	if err != nil {
		log.Error().Msgf("BuildTxTransfer: Failed to open client: %v", err)
		return nil, err
	}

	nonce, err := client.PendingNonceAt(context.Background(), from.Address)
	if err != nil {
		log.Error().Msgf("BuildTxTransfer: Cannot get nonce. Error:(%v)", err)
		return nil, err
	}

	// Suggest gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Error().Msgf("BuildTxTransfer: Cannot suggest gas price. Error:(%v)", err)
		return nil, err
	}

	tx := types.NewTransaction(nonce, to, amount, uint64(21000), gasPrice, data)
	return tx, nil

}

func BuildHailToSendTxTemplate(b *cmn.Blockchain, from *cmn.Address, to common.Address,
	amount *big.Int, data []byte, suggested_gas_price *big.Int) (string, error) {
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

	dollars := ""
	if nt.Price > 0 {
		dollars = cmn.TagShortDollarLink(nt.Price * nt.Float64(amount))
	} else {
		dollars = "(unknown)"
	}

	tx, err := BuildTx(b, s, from, to, amount, data)

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
        From: ` + cmn.AddressShortLinkTag(from.Address) + " " + from.Name + `
          To: ` + cmn.AddressShortLinkTag(to) + " " + to_name + `
      Amount: ` + nt.Value2Str(amount) + " " + nt.Symbol + `
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
