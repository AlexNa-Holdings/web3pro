package cmn

import (
	"github.com/AlexNa-Holdings/web3pro/gocui"
)

type HailRequest struct {
	Priorized      bool
	Title          string
	Template       string
	OnOpen         func(g *gocui.Gui, v *gocui.View)
	OnClose        func()
	OnSuspend      func()
	OnResume       func()
	OnClickHotspot func(v *gocui.View, hs *gocui.Hotspot)
	Done           chan bool
	Suspended      bool
}

var HailChannel = make(chan *HailRequest)

func Hail(request *HailRequest) {
	HailChannel <- request
}

func HailAndWait(request *HailRequest) {
	HailChannel <- request
	<-request.Done
}
