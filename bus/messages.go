package bus

import (
	"math/big"
	"time"

	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// ---------- timer ----------
type B_TimerInit struct { // init
	Limit     time.Duration
	HardLimit time.Duration
	Start     bool
}

type B_TimerInitHard struct { // init-hard
	TimerId   int
	Limit     time.Duration
	HardLimit time.Duration
	Start     bool
}

type B_TimerTick struct { // tick
	Tick int
	Left map[int]time.Duration // id -> seconds left
}

// ---------- ui ----------

// string // command

type B_Hail struct { // hail
	Priorized      bool
	Title          string
	Template       string
	OnOpen         func(*Message, *gocui.Gui, *gocui.View)
	OnClose        func(*Message)
	OnCancel       func(*Message)
	OnOk           func(*Message)
	OnSuspend      func(*Message)
	OnResume       func(*Message)
	OnTick         func(*Message, int)
	OnClickHotspot func(*Message, *gocui.View, *gocui.Hotspot)
	OnOverHotspot  func(*Message, *gocui.View, *gocui.Hotspot)
	Suspended      bool
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

type B_UsbIsConnected struct { // is-connected
	USB_ID string
}

type B_UsbIsConnected_Response struct { // is-connected_response
	Connected bool
}

// ---------- signer ----------
type B_SignerGetAddresses struct { // get-addresses
	Type      string
	Name      string
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
	Name string
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

type B_SignerConnected struct { // connected
	Type string
	Name string
}

type B_SignerSignTx struct { // sign-tx
	Type      string
	Name      string
	MasterKey string
	Chain     string
	Tx        *types.Transaction
	From      common.Address
	Path      string
}

type B_SignerSignTypedData_v4 struct { // sign-typed-data-v4
	Type      string
	Name      string
	MasterKey string
	Address   common.Address
	Path      string
	TypedData apitypes.TypedData
}

// ---------- ws ----------
type B_WsList_Conn struct { // device
	Agent string
}
type B_WsList_Response []B_WsList_Conn

// ---------- eth ----------
type B_EthSend struct { // send
	Blockchain string
	Token      string
	From       common.Address
	To         common.Address
	Amount     *big.Int
}

type B_EthSendTx struct { // send
	Blockchain string
	From       common.Address
	To         common.Address
	Amount     *big.Int
	Data       []byte
}

type B_EthSignTypedData_v4 struct { // sign-typed-data-v4
	Blockchain string
	Address    common.Address
	TypedData  apitypes.TypedData
}
