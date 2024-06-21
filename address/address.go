package address

import "github.com/ethereum/go-ethereum/common"

type Address struct {
	Name       string         `json:"name"`
	Address    common.Address `json:"address"`
	SignerType string         `json:"signer_type"`
	SignerSN   string         `json:"signer_sn"`
	Path       string         `json:"path"`
}
