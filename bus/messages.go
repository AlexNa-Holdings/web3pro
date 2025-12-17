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
	OnOk           func(*Message, *gocui.View) bool // return true to close hail
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

type B_SignerSign struct { // sign-typed-data-v4
	Type      string
	Name      string
	MasterKey string
	Address   common.Address
	Path      string
	Data      []byte
}

// ---------- ws ----------
type B_WsList_Conn struct { // device
	Agent string
}
type B_WsList_Response []B_WsList_Conn

// ---------- eth ----------
type B_EthSend struct { // send
	ChainId int
	Token   string
	From    common.Address
	To      common.Address
	Amount  *big.Int
}

type B_EthSendTx struct { // send
	ChainId int
	From    common.Address
	To      common.Address
	Amount  *big.Int
	Data    []byte
}

type B_EthCall struct { // send
	ChainId int
	From    common.Address
	To      common.Address
	Amount  *big.Int
	Data    []byte
}

type B_EthMultiCall_Call struct {
	To   common.Address
	Data []byte
}

type B_EthMultiCall struct { // send
	ChainId int
	From    common.Address
	Amount  *big.Int
	Calls   []B_EthMultiCall_Call
}

type B_EthSignTypedData_v4 struct { // sign-typed-data-v4
	Blockchain string
	Address    common.Address
	TypedData  apitypes.TypedData
}

type B_EthSign struct { // sign
	Blockchain string
	Address    common.Address
	Data       []byte
}

type B_EthEstimateGas struct { // estimate-gas
	Blockchain string
	From       common.Address
	To         common.Address
	Amount     *big.Int
	Data       []byte
}

type B_EthBlockNumber struct { // get-block-number
	Blockchain string
}

type B_EthTxByHash struct { // get-tx-by-hash
	Blockchain string
	Hash       common.Hash
}

type B_EthTxByHash_Response struct { // get-tx-by-hash_response
	BlockHash        string `json:"blockHash"`
	BlockNumber      string `json:"blockNumber"`
	ChainID          int    `json:"chainId"`
	From             string `json:"from"`
	Gas              string `json:"gas"`
	GasPrice         string `json:"gasPrice"`
	Hash             string `json:"hash"`
	Input            string `json:"input"`
	Nonce            string `json:"nonce"`
	To               string `json:"to"`
	TransactionIndex string `json:"transactionIndex"`
	Value            string `json:"value"`
	V                string `json:"v"`
	R                string `json:"r"`
	S                string `json:"s"`
}

// ---------- explorer ----------
type B_ExplorerDownloadContract struct { // download-contract
	Blockchain string
	Address    common.Address
}

// ---------- lp_v2 ----------
type B_LP_V2_Discover struct { // discover
	ChainId int
	Name    string
	Token0  common.Address // Optional: filter pairs by token0 address
	Token1  common.Address // Optional: filter pairs by token1 address
}

type B_LP_V2_GetPair struct { // get-pair
	ChainId int
	Factory common.Address
	Token0  common.Address
	Token1  common.Address
}

type B_LP_V2_GetReserves struct { // get-reserves
	ChainId int
	Pair    common.Address
}

type B_LP_V2_GetReserves_Response struct { // get-reserves_response
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}

type B_LP_V2_GetPositionStatus struct { // get-position-status
	ChainId int
	Factory common.Address
	Pair    common.Address
}

type B_LP_V2_GetPositionStatus_Response struct { // get-position-status_response
	Owner             common.Address
	ChainId           int
	Token0            common.Address
	Token1            common.Address
	Factory           common.Address
	Pair              common.Address
	LPBalance         *big.Int
	TotalSupply       *big.Int
	Reserve0          *big.Int
	Reserve1          *big.Int
	Liquidity0        *big.Int
	Liquidity1        *big.Int
	Liquidity0Dollars float64
	Liquidity1Dollars float64
	ProviderName      string
}

// ---------- lp_v3 ----------
type B_LP_V3_Discover struct { // discover
	ChainId int
	Name    string
}

type B_LP_V3_GetNftPosition struct { // get-nft-position
	ChainId   int
	Provider  common.Address
	From      common.Address
	NFT_Token *big.Int
}

type B_LP_V3_GetNftPosition_Response struct { // get-nft-position_response
	Nonce                                              *big.Int
	Operator                                           common.Address
	Token0                                             common.Address
	Token1                                             common.Address
	Fee                                                *big.Int
	TickLower, TickUpper                               int64
	Liquidity                                          *big.Int
	FeeGrowthInside0LastX128, FeeGrowthInside1LastX128 *big.Int
	TokensOwed0, TokensOwed1                           *big.Int
}

type B_LP_V3_GetPoolPosition struct { // get-pool-position
	ChainId              int
	Provider             common.Address
	Pool                 common.Address
	TickLower, TickUpper int64
}

type B_LP_V3_GetPoolPosition_Response struct { // get-pool-position_response
	Liquidity                                          *big.Int
	FeeGrowthInside0LastX128, FeeGrowthInside1LastX128 *big.Int
	TokensOwed0, TokensOwed1                           *big.Int
}

type B_LP_V3_GetFactory struct { // get-factory
	ChainId  int
	Provider common.Address
}

type B_LP_V3_GetPool struct { // get-pool
	ChainId  int
	Provider common.Address
	Factory  common.Address
	Token0   common.Address
	Token1   common.Address
	Fee      *big.Int
}

type B_LP_V3_GetSlot0 struct { // get-price
	ChainId int
	Pool    common.Address
}

type B_LP_V3_GetSlot0_Response struct { // get-price_response
	SqrtPriceX96 *big.Int
	Tick         int64
	FeeProtocol0 float32 // percentage
	FeeProtocol1 float32 // percentage
	Unlocked     bool
}

type B_LP_V3_GetFeeGrowth struct { // get-fee-grows
	ChainId int
	Pool    common.Address
}

type B_LP_V3_GetFeeGrowth_Response struct { // get-fee-grows_response
	FeeGrowthGlobal0X128, FeeGrowthGlobal1X128 *big.Int
}

type B_LP_V3_GetTick struct { // get-tick
	ChainId int
	Pool    common.Address
	Tick    int64
}

type B_LP_V3_GetTick_Response struct { // get-tick_response
	LiquidityGross, LiquidityNet                 *big.Int
	FeeGrowthOutside0X128, FeeGrowthOutside1X128 *big.Int
	TickCumulativeOutside                        *big.Int
	SecondsPerLiquidityOutsideX128               *big.Int
	SecondsOutside                               uint32
	Initialized                                  bool
}

type B_LP_V3_GetPositionStatus struct { // get-position-status
	ChainId   int
	Provider  common.Address
	NFT_Token *big.Int
}

type B_LP_V3_GetPositionStatus_Response struct { // get-position-status_response
	Owner             common.Address
	ChainId           int
	Token0            common.Address
	Token1            common.Address
	Provider          common.Address
	On                bool
	Liquidity0        *big.Int
	Liquidity1        *big.Int
	Liquidity0Dollars float64
	Liquidity1Dollars float64
	Gain0             *big.Int
	Gain1             *big.Int
	Gain0Dollars      float64
	Gain1Dollars      float64
	ProviderName      string
	FeeProtocol0      float32 // percentage
	FeeProtocol1      float32 // percentage
}

// ---------- lp_v4 ----------
type B_LP_V4_Discover struct { // discover
	ChainId int
	Name    string
	TokenId *big.Int // Optional: specific NFT token ID to add
}

type B_LP_V4_GetPositionStatus struct { // get-position-status
	ChainId   int
	Provider  common.Address
	NFT_Token *big.Int
}

type B_LP_V4_GetPositionStatus_Response struct { // get-position-status_response
	Owner             common.Address
	ChainId           int
	NFT_Token         *big.Int
	Currency0         common.Address
	Currency1         common.Address
	Provider          common.Address
	PoolManager       common.Address
	PoolId            [32]byte
	TickLower         int64
	TickUpper         int64
	On                bool
	Fee               int64
	Liquidity         *big.Int
	Liquidity0        *big.Int
	Liquidity1        *big.Int
	Liquidity0Dollars float64
	Liquidity1Dollars float64
	Gain0             *big.Int
	Gain1             *big.Int
	Gain0Dollars      float64
	Gain1Dollars      float64
	ProviderName      string
	HookAddress       common.Address
}

type B_LP_V4_GetNftPosition struct { // get-nft-position
	ChainId   int
	Provider  common.Address
	From      common.Address
	NFT_Token *big.Int
}

type B_LP_V4_GetNftPosition_Response struct { // get-nft-position_response
	PoolId      [32]byte
	TickLower   int64
	TickUpper   int64
	Liquidity   *big.Int
	Currency0   common.Address
	Currency1   common.Address
	Fee         int64
	TickSpacing int64
	HookAddress common.Address
}

// ---------- staking ----------
type B_Staking_GetBalance struct { // get-balance
	ChainId      int
	Contract     common.Address
	Owner        common.Address
	ValidatorId  uint64         // Optional: for validator-based staking (e.g., Monad)
	VaultAddress common.Address // Optional: for vault-based staking (e.g., Aztec)
}

type B_Staking_GetBalance_Response struct { // get-balance_response
	Balance       *big.Int
	StakedPercent float64 // Optional: for vault-based staking, percentage of balance that is staked (0-100)
}

type B_Staking_GetPending struct { // get-pending
	ChainId      int
	Contract     common.Address
	Owner        common.Address
	RewardToken  common.Address
	ValidatorId  uint64         // Optional: for validator-based staking (e.g., Monad)
	VaultAddress common.Address // Optional: for vault-based staking (e.g., Aztec)
}

type B_Staking_GetPending_Response struct { // get-pending_response
	Pending *big.Int
}

type B_Staking_GetDelegations struct { // get-delegations
	ChainId  int
	Contract common.Address // Staking precompile address
	Owner    common.Address
}

type B_Staking_GetDelegations_Response struct { // get-delegations_response
	ValidatorIds []uint64
}
