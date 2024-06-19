package blockchain

var PrefefinedBlockchains []Blockchain = []Blockchain{
	{
		Name:        "Ethereum",
		Url:         "https://mainnet.infura.io/v3/your_infura_key",
		ChainId:     1,
		ExplorerUrl: "https://etherscan.io",
		Currency:    "ETH",
	},
	{
		Name:        "PulseChain",
		Url:         "wss://rpc.pulsechain.com",
		ChainId:     369,
		ExplorerUrl: "https://pulsechain.com/explorer",
		Currency:    "PLS",
	},

	// test chains
	{
		Name:        "PulseChain Testnet v4",
		Url:         "https://rpc.v4.testnet.pulsechain.com",
		ChainId:     943,
		ExplorerUrl: "https://alfajores-blockscout.celo-testnet.org",
	},
}
