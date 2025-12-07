package cmn

import (
	"math/big"
	"sync"
	"time"

	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
)

var StandardOnClickHotspot func(v *gocui.View, hs *gocui.Hotspot)
var StandardOnOverHotspot func(v *gocui.View, hs *gocui.Hotspot)

const (
	ICON_DELETE   = "\U0000f057 " //"\uf00d"
	ICON_EDIT     = "\uf044 "
	ICON_COPY     = "\uf0c5 "
	ICON_DROPLIST = "\ueb6e "
	ICON_PROMOTE  = "\ued65 "
	ICON_ADD      = "\ueadc "
	ICON_3DOTS    = "\U000f01d8"
	ICON_BACK     = "\U000f006e "
	ICON_SEND     = "\U000f048a "
	ICON_LINK     = "\uf08e "
	ICON_FEED     = "\uf09e "
	ICON_DOWNLOAD = "\ueac2 "
	ICON_VSC      = "\U000f0a1e "
	ICON_TRUST    = "\uebc1 "
	ICON_NO_ENTRY = "\uf4f4 "
	ICON_LIGHT    = "\U000f06e8 "
	ICON_CHECK    = "\U000f0134 "
	ICON_UNCHECK  = "\U000f0130 "
	ICON_ALERT    = "\U000f0028 "
)

const (
	ALERT_ARROW = "<blink>\U000f0028\uf178 </blink>"
)

type Wallet struct {
	Name            string            `json:"name"`
	Blockchains     []*Blockchain     `json:"blockchains"`
	Signers         []*Signer         `json:"signers"`
	Addresses       []*Address        `json:"addresses"`
	Tokens          []*Token          `json:"tokens"`
	Origins         []*Origin         `json:"origins"`
	LP_V2_Providers []*LP_V2          `json:"lp_v2_providers"`
	LP_V2_Positions []*LP_V2_Position `json:"lp_v2_positions"`
	LP_V3_Providers []*LP_V3          `json:"lp_v3_providers"`
	LP_V3_Positions []*LP_V3_Position `json:"lp_v3_positions"`
	LP_V4_Providers   []*LP_V4            `json:"lp_v4_providers"`
	LP_V4_Positions   []*LP_V4_Position   `json:"lp_v4_positions"`
	Stakings          []*Staking          `json:"stakings"`
	StakingPositions  []*StakingPosition  `json:"staking_positions"`
	Contracts         map[common.Address]*Contract
	AppsPaneOn      bool `json:"apps_pane_on"`
	LP_V2PaneOn     bool `json:"lp_v2_pane_on"`
	LP_V3PaneOn     bool `json:"lp_v3_pane_on"`
	LP_V4PaneOn     bool `json:"lp_v4_pane_on"`
	TokenPaneOn     bool `json:"token_pane_on"`
	StakingPaneOn   bool `json:"staking_pane_on"`

	CurrentChainId int            `json:"current_chain_id"`
	CurrentAddress common.Address `json:"current_address"`
	CurrentOrigin  string         `json:"current_origin"`

	// Auxilary params
	ParamInt map[string]int    `json:"param_int"`
	ParamStr map[string]string `json:"param_str"`

	SoundOn bool   `json:"sound_on"`
	Sound   string `json:"sound"`

	filePath   string     `json:"-"`
	password   string     `json:"-"`
	writeMutex sync.Mutex `json:"-"`
}

type Contract struct {
	Name    string `json:"name"`
	Trusted bool   `json:"trusted"`
}

type Origin struct {
	URL       string           `json:"url"`
	ChainId   int              `json:"chain_id"`
	Addresses []common.Address `json:"addresses"`
}

type Address struct {
	Name    string         `json:"name"`
	Tag     string         `json:"tag"`
	Address common.Address `json:"address"`
	Signer  string         `json:"signer"`
	Path    string         `json:"path"`
}

type Blockchain struct {
	Name             string         `json:"name"`
	ShortName        string         `json:"short_name"` // 4 chars max, for display in LP listings
	Url              string         `json:"url"`
	ChainId          int            `json:"chain_id"`
	ExplorerUrl      string         `json:"explorer_url"`
	ExplorerAPIUrl   string         `json:"explorer_api_url"`
	ExplorerAPIToken string         `json:"explorer_api_token"`
	ExplorerApiType  string         `json:"explorer_api_type"`
	Currency         string         `json:"currency"`
	WTokenAddress    common.Address `json:"wrapped_native_token_address"`
	Multicall        common.Address `json:"multicall"`
	RPCRateLimit     int            `json:"rpc_rate_limit,omitempty"` // RPC calls per second (auto-tuned or fixed)
	RPCRateAuto      bool           `json:"rpc_rate_auto,omitempty"`  // true = auto-tune rate, false = use fixed rate
}

var EXPLORER_API_TYPES = []string{"etherscan", "blockscout"}

var KNOWN_SIGNER_TYPES = []string{"mnemonics", "ledger", "trezor"}

type Token struct {
	ChainId        int            `json:"chain_id"`
	Name           string         `json:"name"`
	Symbol         string         `json:"symbol"`
	Address        common.Address `json:"address"`
	Decimals       int            `json:"decimals"`
	Native         bool           `json:"native"`
	Unique         bool           `json:"-"` // Unique name in the blockchain
	Ignored        bool           `json:"ignored,omitempty"`
	PriceFeeder    string         `json:"price_feeder"`
	PriceFeedParam string         `json:"price_feed_id"`
	Price          float64        `json:"price"`
	PriceChange24  float64        `json:"price_change_24"`
	PriceTimestamp time.Time      `json:"price_timestamp"` // Unix timestamp
}

var KNOWN_FEEDERS = []string{"dexscreener", "coinmarketcap"}

type LP_V2 struct { // LP v2 Provider (e.g., Uniswap V2, SushiSwap)
	Name       string         `json:"name"`
	Factory    common.Address `json:"factory"`     // Factory contract address
	Router     common.Address `json:"router"`      // Router contract address
	ChainId    int            `json:"chain_id"`
	URL        string         `json:"url"`         // Web UI URL
	SubgraphID string         `json:"subgraph_id"` // The Graph subgraph ID for discovery
}

type LP_V2_Position struct {
	Owner     common.Address `json:"owner"`
	ChainId   int            `json:"chain_id"`
	Factory   common.Address `json:"factory"`
	Pair      common.Address `json:"pair"`   // LP token / pair contract address
	Token0    common.Address `json:"token0"`
	Token1    common.Address `json:"token1"`
}

type LP_V3 struct { // LP v3 Position Manager
	Name     string         `json:"name"`
	Provider common.Address `json:"provider"`
	ChainId  int            `json:"chain_id"`
	URL      string         `json:"url"`
}

type LP_V3_Position struct {
	Owner     common.Address `json:"owner"`
	ChainId   int            `json:"chain_id"`
	Provider  common.Address `json:"provider"`
	NFT_Token *big.Int       `json:"nft_token"`
	Token0    common.Address `json:"token0"`
	Token1    common.Address `json:"token1"`
	Fee       *big.Int       `json:"fee"`
	Pool      common.Address `json:"pool"`
	TickLower int64          `json:"tick_lower"`
	TickUpper int64          `json:"tick_upper"`
}

type LP_V4 struct { // LP v4 Position Manager
	Name        string         `json:"name"`
	Provider    common.Address `json:"provider"`     // PositionManager contract
	PoolManager common.Address `json:"pool_manager"` // Singleton PoolManager
	StateView   common.Address `json:"state_view"`   // StateView contract for reading pool state
	ChainId     int            `json:"chain_id"`
	URL         string         `json:"url"`          // Web UI URL
	SubgraphURL string         `json:"subgraph_url"` // Subgraph API URL for discovery
}

type LP_V4_Position struct {
	Owner       common.Address `json:"owner"`
	ChainId     int            `json:"chain_id"`
	Provider    common.Address `json:"provider"`
	PoolManager common.Address `json:"pool_manager"`
	NFT_Token   *big.Int       `json:"nft_token"`
	PoolId      [32]byte       `json:"pool_id"`
	Currency0   common.Address `json:"currency0"`
	Currency1   common.Address `json:"currency1"`
	Fee         int64          `json:"fee"`
	TickSpacing int64          `json:"tick_spacing"`
	TickLower   int64          `json:"tick_lower"`
	TickUpper   int64          `json:"tick_upper"`
	Liquidity   *big.Int       `json:"liquidity"`
	HookAddress common.Address `json:"hook_address"`
}

// Staking represents a staking contract configuration
type Staking struct {
	Name            string         `json:"name"`
	ChainId         int            `json:"chain_id"`
	Contract        common.Address `json:"contract"`       // Staking contract address
	URL             string         `json:"url"`            // Provider website URL
	StakedToken     common.Address `json:"staked_token"`   // Token being staked
	BalanceFunc     string         `json:"balance_func"`   // Function name to get staked balance (e.g., "balanceOf", "stakedAmount")
	Reward1Token    common.Address `json:"reward1_token"`  // First reward token address
	Reward1Func     string         `json:"reward1_func"`   // First reward pending function (e.g., "earned")
	Reward2Token    common.Address `json:"reward2_token"`  // Second reward token address (optional)
	Reward2Func     string         `json:"reward2_func"`   // Second reward pending function (optional)
	ValidatorId     uint64         `json:"validator_id"`   // Validator ID for native staking (e.g., Monad)
	Hardcoded       bool           `json:"hardcoded"`      // If true, uses custom logic (cannot be configured, only deleted)
}

// StakingPosition represents a user's position in a staking contract
type StakingPosition struct {
	Owner       common.Address `json:"owner"`
	ChainId     int            `json:"chain_id"`
	Contract    common.Address `json:"contract"`     // Reference to the staking contract
	ValidatorId uint64         `json:"validator_id"` // Validator ID for native staking (e.g., Monad)
}
