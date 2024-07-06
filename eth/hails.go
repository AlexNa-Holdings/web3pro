package eth

import (
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

func HailToSend(b *cmn.Blockchain, t *cmn.Token, from, to common.Address, amount *big.Int) {

	if wallet.CurrentWallet == nil {
		log.Error().Msg("No wallet")
		return
	}

	w := wallet.CurrentWallet

	from_addr := w.GetAddress(from.String())
	if from_addr == nil {
		log.Error().Msg("Unknown address: " + from.String())
		return
	}

	s := w.GetSigner(from_addr.Signer)
	if s == nil {
		log.Error().Msg("Unknown signer: " + from_addr.Signer)
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

	cmn.Hail(&cmn.HailRequest{
		Title: "Send Tokens",
		Template: `Blockchain: ` + b.Name + `
Token: ` + t.Symbol + tc + `
From: ` + cmn.AddressShortLinkTag(from) + " " + from_addr.Name + `
Signer: ` + s.Name + " (" + s.Type + ")" + `
To: ` + cmn.AddressShortLinkTag(to) + " " + to_name + `
Amount: ` + t.Value2Str(amount) + " " + t.Symbol + `
<c>
<button text:Send bgcolor:g.HelpBgColor color:g.HelpFgColor tip:"send tokens">  <button text:Reject bgcolor:g.ErrorFgColor tip:"reject transaction">
`,
		OnOpen: func(hr *cmn.HailRequest, g *gocui.Gui, v *gocui.View) {
			v.SetInput("to", to.String())
			v.SetInput("amount", amount.String())
		},
		OnClose: func(hr *cmn.HailRequest) {
			log.Debug().Msg("HailToSend closed")
		},
		OnOk: func(hr *cmn.HailRequest) {
		},
		OnOverHotspot: func(hr *cmn.HailRequest, v *gocui.View, hs *gocui.Hotspot) {
			cmn.StandardOnOverHotspot(v, hs)
		},
		OnClickHotspot: func(hr *cmn.HailRequest, v *gocui.View, hs *gocui.Hotspot) {
			cmn.StandardOnClickHotspot(v, hs)
		},
	})

}
