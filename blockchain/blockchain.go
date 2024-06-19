package blockchain

type Blockchain struct {
	Name        string `json:"name"`
	Url         string `json:"url"`
	ChainId     uint   `json:"chain_id"`
	ExplorerUrl string `json:"explorer_url"`
	Currency    string `json:"currency"`
}
