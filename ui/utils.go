package ui

import (
	"fmt"
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

func TagAddressShortLink(a common.Address) string {
	s := a.String()

	return fmt.Sprintf("<l text:'%s%s%s' action:'copy %s' tip:'%s'>",
		s[:6], gocui.ICON_3DOTS, s[len(s)-4:], a.String(), a.String())
}

func AddValueLink(v *gocui.View, val *big.Int, t *cmn.Token) {
	if v == nil {
		return
	}

	if t == nil {
		return
	}

	xf := cmn.NewXF(val, t.Decimals)

	text := cmn.FmtAmount(val, t.Decimals, true)
	v.AddLink(text, "copy "+xf.String(), "Copy "+xf.String(), "")
}

func TagValueLink(val *big.Int, t *cmn.Token) string {
	if t == nil {
		return ""
	}

	xf := cmn.NewXF(val, t.Decimals)

	return fmt.Sprintf("<l text:'%s' action:'copy %s' tip:'%s'>", cmn.FmtAmount(val, t.Decimals, true), xf.String(), xf.String())
}

func AddDollarValueLink(v *gocui.View, val *big.Int, t *cmn.Token) {
	if v == nil {
		return
	}

	if t == nil {
		return
	}

	xf := cmn.NewXF(val, t.Decimals)
	f := t.Price * xf.Float64()

	text := cmn.FmtFloat64D(f, true)
	n := fmt.Sprintf("%f", f)

	v.AddLink(text, "copy "+n, "Copy "+n, "")
}

func TagDollarValueLink(val *big.Int, t *cmn.Token) string {
	if t == nil {
		return ""
	}

	xf := cmn.NewXF(val, t.Decimals)
	f := t.Price * xf.Float64()

	return fmt.Sprintf("<l text:'%s' action:'copy %f' tip:'%f'>", cmn.FmtFloat64D(f, true), f, f)
}

func TagShortDollarValueLink(val *big.Int, t *cmn.Token) string {
	if t == nil {
		return ""
	}

	xf := cmn.NewXF(val, t.Decimals)
	f := t.Price * xf.Float64()

	return fmt.Sprintf("<l text:'%s' action:'copy %f' tip:'%f'>", cmn.FmtFloat64D(f, false), f, f)
}

func TagShortDollarLink(val float64) string {
	return fmt.Sprintf("<l text:'%s' action:'copy %f' tip:'%f'>", cmn.FmtFloat64D(val, false), val, val)
}

func AddValueSymbolLink(v *gocui.View, val *big.Int, t *cmn.Token) {
	if v == nil {
		return
	}

	if t == nil {
		return
	}

	xf := cmn.NewXF(val, t.Decimals)
	text := cmn.FmtAmount(val, t.Decimals, true)

	v.AddLink(text, "copy "+xf.String(), "Copy "+xf.String(), "")
}

func TagValueSymbolLink(val *big.Int, t *cmn.Token) string {
	if t == nil {
		return ""
	}

	xf := cmn.NewXF(val, t.Decimals)

	return fmt.Sprintf("<l text:'%s' action:'copy %s' tip:'%s'> %s",
		cmn.FmtAmount(val, t.Decimals, true), xf.String(), xf.String(), t.Symbol)
}

func TagShortValueSymbolLink(val *big.Int, t *cmn.Token) string {
	if t == nil {
		return ""
	}

	xf := cmn.NewXF(val, t.Decimals)

	return fmt.Sprintf("<l text:'%s' action:'copy %s' tip:'%s'> %s",
		cmn.FmtAmount(val, t.Decimals, false), xf.String(), xf.String(), t.Symbol)
}
