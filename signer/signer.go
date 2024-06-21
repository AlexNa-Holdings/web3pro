package signer

type SignerDriver interface {
}

type Signer struct {
	Name string            `json:"name"`
	Type string            `json:"type"`
	SN   string            `json:"sn"`
	P    map[string]string `json:"params"`
}

var KNOWN_SIGNER_TYPES = []string{"trezor", "ledger", "mnemonic"}

func GetType(manufacturer string, product string) string {
	if product == "TREZOR" {
		return "trezor"
	}

	if manufacturer == "Ledger" {
		return "ledger"
	}

	return ""
}
