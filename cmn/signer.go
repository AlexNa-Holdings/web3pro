package cmn

type Signer struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	MasterKey string   `json:"master-key"`
	Copies    []string `json:"copies"`
}

var STANDARD_DERIVATIONS = map[string]struct {
	Name   string
	Format string
}{
	"legacy": {
		Name:   "Legacy (MEW, MyCrypto) m/44'/60'/0'/%d",
		Format: "m/44'/60'/0'/%d",
	},
	"ledger-live": {
		Name:   "Ledger Live m/44'/60'/%d'/0/0",
		Format: "m/44'/60'/%d'/0/0",
	},
	"default": {
		Name:   "Default m/44'/60'/0'/0/%d",
		Format: "m/44'/60'/0'/0/%d",
	},
}

func (s *Signer) GetAllCopies() []string {
	r := []string{s.Name}
	r = append(r, s.Copies...)
	return r
}
