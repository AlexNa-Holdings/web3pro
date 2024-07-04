package cmn

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Signer struct {
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	SN     string   `json:"sn"`
	Copies []string `json:"copies"`
}

type SignerDriver interface {
	IsConnected(signer *Signer) bool
	GetName(path string) (string, error) // only for HW wallets
	GetAddresses(signer *Signer, path string, start_from int, count int) ([]Address, error)
	PrintDetails(path string) string
}

var WalletTrezorDriver SignerDriver
var WalletMnemonicsDriver SignerDriver

type Address struct {
	Name    string         `json:"name"`
	Tag     string         `json:"tag"`
	Address common.Address `json:"address"`
	Signer  string         `json:"signer"`
	Path    string         `json:"path"`
}

type Blockchain struct {
	Name        string `json:"name"`
	Url         string `json:"url"`
	ChainId     uint   `json:"chain_id"`
	ExplorerUrl string `json:"explorer_url"`
	Currency    string `json:"currency"`

	Client *ethclient.Client `json:"-"`
}

type Token struct {
	Blockchain string         `json:"blockchain"`
	Name       string         `json:"name"`
	Symbol     string         `json:"symbol"`
	Address    common.Address `json:"address"`
	Decimals   int            `json:"decimals"`
	Native     bool           `json:"native"`
}
