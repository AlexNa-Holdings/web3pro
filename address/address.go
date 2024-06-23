package address

import "github.com/ethereum/go-ethereum/common"

type Address struct {
	Name    string         `json:"name"`
	Address common.Address `json:"address"`
	Signer  string         `json:"signer"`
	Path    string         `json:"path"`
}