package cmn

import (
	"sync"
	"time"

	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var StandardOnClickHotspot func(v *gocui.View, hs *gocui.Hotspot)
var StandardOnOverHotspot func(v *gocui.View, hs *gocui.Hotspot)

type Wallet struct {
	Name           string         `json:"name"`
	Blockchains    []*Blockchain  `json:"blockchains"`
	Signers        []*Signer      `json:"signers"`
	Addresses      []*Address     `json:"addresses"`
	Tokens         []*Token       `json:"tokens"`
	Origins        []*Origin      `json:"origins"`
	CurrentChain   string         `json:"current_chain"`
	CurrentAddress common.Address `json:"current_address"`

	filePath   string     `json:"-"`
	password   string     `json:"-"`
	writeMutex sync.Mutex `json:"-"`
}

type Origin struct {
	URL       string           `json:"url"`
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
	Name          string         `json:"name"`
	Url           string         `json:"url"`
	ChainId       uint           `json:"chain_id"`
	ExplorerUrl   string         `json:"explorer_url"`
	Currency      string         `json:"currency"`
	WTokenAddress common.Address `json:"wrapped_native_token_address"`

	Client *ethclient.Client `json:"-"`
}

var KNOWN_SIGNER_TYPES = []string{"mnemonics", "ledger", "trezor"}

type Token struct {
	Blockchain     string         `json:"blockchain"`
	Name           string         `json:"name"`
	Symbol         string         `json:"symbol"`
	Address        common.Address `json:"address"`
	Decimals       int            `json:"decimals"`
	Native         bool           `json:"native"`
	Unique         bool           `json:"-"` // Unique name in the blockchain
	PriceFeeder    string         `json:"price_feeder"`
	PriceFeedParam string         `json:"price_feed_id"`
	Price          float64        `json:"price"`
	PraceChange24  float64        `json:"price_change_24"`
	PriceTimestamp time.Time      `json:"price_timestamp"` // Unix timestamp
}
