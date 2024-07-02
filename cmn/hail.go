package cmn

import (
	"time"

	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

type HailRequest struct {
	Priorized      bool
	Title          string
	Template       string
	OnOpen         func(hr *HailRequest, g *gocui.Gui, v *gocui.View)
	OnClose        func(hr *HailRequest)
	OnCancel       func(hr *HailRequest)
	OnOk           func(hr *HailRequest)
	OnSuspend      func(hr *HailRequest)
	OnResume       func(hr *HailRequest)
	OnTick         func(hr *HailRequest, tick int)
	OnClickHotspot func(hr *HailRequest, v *gocui.View, hs *gocui.Hotspot)
	Done           chan bool
	Suspended      bool
	TimeoutSec     int // in seconds

	// Internal
	Expiration time.Time
}

var HailChannel = make(chan *HailRequest)
var RemoveHailChannel = make(chan *HailRequest, 10)

func Hail(hail *HailRequest) {
	HailChannel <- hail
}

func HailAndWait(hail *HailRequest) {
	log.Trace().Msgf("Hail & Wait: %s", hail.Title)
	HailChannel <- hail
	<-hail.Done
}

func (hail *HailRequest) Close() {
	RemoveHailChannel <- hail
}
