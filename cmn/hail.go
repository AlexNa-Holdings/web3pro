package cmn

import (
	"time"

	"github.com/AlexNa-Holdings/web3pro/gocui"
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
	OnOverHotspot  func(hr *HailRequest, v *gocui.View, hs *gocui.Hotspot)
	Done           chan bool
	Suspended      bool
	TimeoutSec     int // in seconds
	TimerPaused    bool

	// Internal
	Expiration time.Time
}

var HailChannel = make(chan *HailRequest)
var RemoveHailChannel = make(chan *HailRequest, 10)

func Hail(hail *HailRequest) {
	hail.Done = make(chan bool, 10)
	HailChannel <- hail
}

func HailAndWait(hail *HailRequest) {
	Hail(hail)
	<-hail.Done
	close(hail.Done)
}

func (hail *HailRequest) Close() {
	RemoveHailChannel <- hail
}
