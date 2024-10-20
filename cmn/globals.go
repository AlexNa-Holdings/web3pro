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

type Wallet struct {
	Name            string            `json:"name"`
	Blockchains     []*Blockchain     `json:"blockchains"`
	Signers         []*Signer         `json:"signers"`
	Addresses       []*Address        `json:"addresses"`
	Tokens          []*Token          `json:"tokens"`
	Origins         []*Origin         `json:"origins"`
	LP_V3_Providers []*LP_V3          `json:"lp_v3_providers"`
	LP_V3_Positions []*LP_V3_Position `json:"lp_v3_positions"`
	Contracts       map[common.Address]*Contract
	AppsPaneOn      bool `json:"apps_pane_on"`
	LP_V3PaneOn     bool `json:"lp_v3_pane_on"`

	CurrentChain   string         `json:"current_chain"` // TODO delete
	CurrentChainId int            `json:"current_chain_id"`
	CurrentAddress common.Address `json:"current_address"`
	CurrentOrigin  string         `json:"current_origin"`

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
	Url              string         `json:"url"`
	ChainId          int            `json:"chain_id"`
	ExplorerUrl      string         `json:"explorer_url"`
	ExplorerAPIUrl   string         `json:"explorer_api_url"`
	ExplorerAPIToken string         `json:"explorer_api_token"`
	ExplorerApiType  string         `json:"explorer_api_type"`
	Currency         string         `json:"currency"`
	WTokenAddress    common.Address `json:"wrapped_native_token_address"`
	Multicall        common.Address `json:"multicall"`
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
	PriceFeeder    string         `json:"price_feeder"`
	PriceFeedParam string         `json:"price_feed_id"`
	Price          float64        `json:"price"`
	PriceChange24  float64        `json:"price_change_24"`
	PriceTimestamp time.Time      `json:"price_timestamp"` // Unix timestamp
}

var KNOWN_FEEDERS = []string{"dexscreener", "coinmarketcap"}

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
