package bus

import (
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
)

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
	USB_ID    string
	Path      string
	Vendor    string
	VendorID  uint16
	Product   string
	ProductID uint16
	Connected bool
}

type B_UsbList_Response []B_UsbList_Device
type B_UsbWrite struct { // write
	USB_ID string
	Data   []byte
}

type B_UsbRead struct { // read
	USB_ID string
}

type B_UsbRead_Response struct { // read_response
	Data []byte
}

type B_UsbConnected struct { // connected
	USB_ID  string
	Vendor  string
	Product string
}

type B_UsbDisconnected struct { // disconnected
	USB_ID string
}

// ---------- signer ----------
type B_SignerGetAddresses struct { // get-addresses
	Type      string
	Name      []string
	MasterKey string
	Path      string
	StartFrom int
	Count     int
}

type B_SignerGetAddresses_Response struct { // get-addresses_response
	Addresses []common.Address
	Paths     []string
}

type B_SignerIsConnected struct { // is-connected
	Type string
	Name []string
}

type B_SignerIsConnected_Response struct { // is-connected_response
	Connected bool
}

type B_SignerList struct { // list
	Type string
}

type B_SignerList_Response struct { // list_response
	Names []string
}
