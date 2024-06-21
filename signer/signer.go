package signer

type SignerDriver interface {
}

type Signer struct {
	Name string            `json:"name"`
	Type string            `json:"type"`
	SN   string            `json:"sn"`
	P    map[string]string `json:"params"`
}

var KNOWN_SIGNER_TYPES = []string{"trezor"}

func GetType(manufacturer string, product string) string {
	if product == "TREZOR" {
		return "trezor"
	}

	if manufacturer == "Ledger" {
		return "ledger"
	}

	return ""
}

// func NewSigner(data *SignerDevice) (Signer, error) {

// 	switch data.Type {
// 	case "trezor":
// 		return NewTrezorSigner(data)
// 	}

// 	return nil, errors.New("unknown signer type: " + data.Type)
// }
