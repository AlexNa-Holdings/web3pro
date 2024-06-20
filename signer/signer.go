package signer

import "errors"

type Signer interface {
}

type SignerData struct {
	Type string            `json:"type"`
	Name string            `json:"name"`
	P    map[string]string `json:"params"`
}

var KNOWN_SIGNER_TYPES = []string{"trezor"}

func NewSigner(data *SignerData) (Signer, error) {

	switch data.Type {
	case "trezor":
		return NewTrezorSigner(data)
	}

	return nil, errors.New("unknown signer type: " + data.Type)
}
