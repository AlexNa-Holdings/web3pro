package eth

import (
	"errors"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

func sign(msg *bus.Message) (string, error) {
	req, ok := msg.Data.(*bus.B_EthSign)
	if !ok {
		return "", bus.ErrInvalidMessageData
	}

	w := cmn.CurrentWallet
	if w == nil {
		return "", fmt.Errorf("no wallet found")
	}

	a := w.GetAddress(req.Address.Hex())
	if a == nil {
		return "", fmt.Errorf("address not found in wallet")
	}

	signer := w.GetSigner(a.Signer)
	if signer == nil {
		return "", fmt.Errorf("signer not found")
	}

	OK := false
	var err error
	var sign string

	msg.Fetch("ui", "hail", &bus.B_Hail{
		Title: "Sign Message",
		Template: `<w>
 Message: ` + string(req.Data) + `
 
<c> <button text:"OK" id:"ok"> <button text:"Cancel" id:"cancel">
`,
		OnOk: func(m *bus.Message, v *gocui.View) bool {

			hail, ok := m.Data.(*bus.B_Hail)
			if !ok {
				log.Error().Msg("sendTx: hail data not found")
				err = errors.New("hail data not found")
				return true
			}

			hail.Template = `<w>
 Message: ` + string(req.Data) + `
 
<c><blink>Waiting to be signed</blink>
<button text:Reject id:cancel bgcolor:g.ErrorFgColor tip:"reject transaction">
`

			v.GetGui().UpdateAsync(func(*gocui.Gui) error {
				hail, ok := m.Data.(*bus.B_Hail)
				if ok {
					v.RenderTemplate(hail.Template)
				}
				return nil
			})

			res := msg.Fetch("signer", "sign", &bus.B_SignerSign{
				Type:      signer.Type,
				Name:      signer.Name,
				MasterKey: signer.MasterKey,
				Address:   a.Address,
				Path:      a.Path,
				Data:      req.Data,
			})
			if res.Error != nil {
				err = res.Error
				return true
			}

			sign = res.Data.(string)
			err = nil
			OK = true
			return true
		},
		OnOverHotspot: func(m *bus.Message, v *gocui.View, hs *gocui.Hotspot) {
			cmn.StandardOnOverHotspot(v, hs)
		},
		OnClickHotspot: func(m *bus.Message, v *gocui.View, hs *gocui.Hotspot) {
			cmn.StandardOnClickHotspot(v, hs)
		},
	})

	if err != nil {
		return "", err
	}

	if !OK {
		return "", fmt.Errorf("cancelled")
	}

	return sign, nil

}
