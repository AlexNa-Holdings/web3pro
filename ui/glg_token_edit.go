package ui

import (
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
)

// name == ""  mreans add new custom blockchain
func DlgTokenEdit(t *cmn.Token) *gocui.Popup {

	w := cmn.CurrentWallet

	if w == nil {
		Notification.ShowError("No wallet open")
		return nil
	}

	b := w.GetBlockchain(t.ChainId)
	if b == nil {
		Notification.ShowError("Blockchain not found")
		return nil
	}

	ta := "(native)"
	if !t.Native {
		ta = t.Address.String()
	}

	return &gocui.Popup{
		Title:         "Edit Token",
		OnOverHotspot: cmn.StandardOnOverHotspot,
		OnOpen: func(v *gocui.View) {
			v.SetInput("name", t.Name)
			v.SetInput("price_feed_param", t.PriceFeedParam)
			v.SetSelectList("price_feeder", cmn.KNOWN_FEEDERS)
			v.SetInput("price_feeder", t.PriceFeeder)

		},
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {

			if hs != nil {
				switch hs.Value {
				case "button Ok":

					t.Name = v.GetInput("name")
					t.PriceFeedParam = v.GetInput("price_feed_param")
					t.PriceFeeder = v.GetInput("price_feeder")

					err := w.Save()
					if err != nil {
						bus.Send("ui", "notify-error", fmt.Sprintf("Error saving wallet: %v", err))
						return
					}
					Gui.HidePopup()
				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: fmt.Sprintf(`
              Name: <input id:name size:32 value:""> 
            Symbol: %s
        Blockchain: %s (%d)
        Token Addr: %s
<line text:Price>
          Provider: <select id:price_feeder size:16>
           Feed ID: <input id:price_feed_param size:32 value:"">
     Current Price: $%s
      Last Updated: %s

 <c><button text:Ok tip:"create wallet">  <button text:Cancel>`,
			t.Symbol, b.Name, t.ChainId, ta, cmn.FmtFloat64(t.Price, false), t.PriceTimestamp.Format("2006-01-02 15:04:05")),
	}
}
