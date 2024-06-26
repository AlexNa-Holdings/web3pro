package cmn

import (
	"time"

	"github.com/AlexNa-Holdings/web3pro/gocui"
)

type HailRequest struct {
	Priorized      bool
	Title          string
	Template       string
	OnOpen         func(g *gocui.Gui, v *gocui.View)
	OnClose        func()
	OnCancel       func()
	OnSuspend      func()
	OnResume       func()
	OnClickHotspot func(v *gocui.View, hs *gocui.Hotspot)
	Done           chan bool
	Suspended      bool
	TimeoutSec     int // in seconds

	// Internal
	Expiration time.Time
}

var HailChannel = make(chan *HailRequest)

func Hail(request *HailRequest) {

	if request.TimeoutSec == 0 {
		request.TimeoutSec = Config.TimeoutSec
	}

	HailChannel <- request
}
