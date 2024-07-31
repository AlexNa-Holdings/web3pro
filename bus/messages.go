package bus

import "github.com/AlexNa-Holdings/web3pro/gocui"

// ---------- timer ----------
type B_TimerInit struct { // init
	LimitSeconds     int
	HardLimitSeconds int
	Start            bool
}

type B_TimerStart struct { // start
	ID int
}

type B_TimerReset struct { // reset
	ID int
}

type B_TimerDone struct { // done
	ID int
}

type B_TimerDelete struct { // delete
	ID int
}
type B_TimerTick struct { // tick
	Tick int
	Left map[int]int // id -> seconds left
}

type B_TimerPause struct { // pause
	ID int
}

// ---------- ui ----------

// string // command

type B_Hail struct { // hail
	Priorized      bool
	Title          string
	Template       string
	OnOpen         func(hr *B_Hail, g *gocui.Gui, v *gocui.View)
	OnClose        func(hr *B_Hail)
	OnCancel       func(hr *B_Hail)
	OnOk           func(hr *B_Hail)
	OnSuspend      func(hr *B_Hail)
	OnResume       func(hr *B_Hail)
	OnTick         func(hr *B_Hail, tick int)
	OnClickHotspot func(hr *B_Hail, v *gocui.View, hs *gocui.Hotspot)
	OnOverHotspot  func(hr *B_Hail, v *gocui.View, hs *gocui.Hotspot)
	Suspended      bool
	TimeoutSec     int // in seconds
	TimerPaused    bool
}

// ---------- usb ----------

type B_UsbList_Device struct { // device
	Path string
	Name string
	Type string
}

type B_UsbList_Response []B_UsbList_Device
