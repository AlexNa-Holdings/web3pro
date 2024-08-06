package ui

import (
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
)

func AddAddressLink(v *gocui.View, a common.Address) {
	if v == nil {
		v = Terminal.Screen
	}

	v.AddLink(a.String(), "copy "+a.String(), "Copy address", "")
}

func AddAddressShortLink(v *gocui.View, a common.Address) {
	if v == nil {
		v = Terminal.Screen
	}

	s := a.String()
	v.AddLink(s[:6]+gocui.ICON_3DOTS+s[len(s)-4:], "copy "+a.String(), "Copy "+a.String(), "")
}

func AddValueLink(v *gocui.View, val *big.Int, t *cmn.Token) {
	if v == nil {
		return
	}

	if t == nil {
		return
	}

	text := cmn.FmtAmount(val, t.Decimals, true)
	n := cmn.Amount2Str(val, t.Decimals)

	v.AddLink(text, "copy "+n, "Copy "+n, "")
}

func AddValueSymbolLink(v *gocui.View, val *big.Int, t *cmn.Token) {
	if v == nil {
		return
	}

	if t == nil {
		return
	}

	text := cmn.FmtAmount(val, t.Decimals, true) + t.Symbol
	n := cmn.Amount2Str(val, t.Decimals)

	v.AddLink(text, "copy "+n, "Copy "+n, "")
}
