package cmn

import "github.com/ethereum/go-ethereum/common"

type PD_V3 struct {
	Name    string
	Address common.Address
	URL     string
}

var PrefedinedLP_V3 = map[int]([]PD_V3){
	1: { // Ethereum Mainnet
		{
			Name:    "Uniswap",
			Address: common.HexToAddress("0xC36442b4a4522E871399CD717aBDD847Ab11FE88"),
			URL:     "https://app.uniswap.org/positions",
		},
		{
			Name:    "PancakeSwap",
			Address: common.HexToAddress("0x46A15B0b27311cedF172AB29E4f4766fbE7F4364"),
			URL:     "https://pancakeswap.finance/liquidity",
		},
		{
			Name:    "SushiSwap",
			Address: common.HexToAddress("0x2214A42d8e2A1d20635c2cb0664422c528B6A432"),
			URL:     "https://www.sushi.com/pool",
		},
	},
	42161: { // Arbitrum One
		{
			Name:    "Uniswap",
			Address: common.HexToAddress("0xC36442b4a4522E871399CD717aBDD847Ab11FE88"),
			URL:     "https://app.uniswap.org/positions",
		},
		{
			Name:    "PancakeSwap",
			Address: common.HexToAddress("0x46A15B0b27311cedF172AB29E4f4766fbE7F4364"),
			URL:     "https://pancakeswap.finance/liquidity",
		},
		{
			Name:    "SushiSwap",
			Address: common.HexToAddress("0x2214A42d8e2A1d20635c2cb0664422c528B6A432"),
			URL:     "https://www.sushi.com/pool",
		},
		{
			Name:    "Camelot",
			Address: common.HexToAddress("0x00c7f3082833e796A5b3e4Bd59f6642FF44DCD15"),
			URL:     "https://app.camelot.exchange/liquidity",
		},
	},
	10: { // Optimism
		{
			Name:    "Uniswap",
			Address: common.HexToAddress("0xC36442b4a4522E871399CD717aBDD847Ab11FE88"),
			URL:     "https://app.uniswap.org/positions",
		},
		{
			Name:    "Velodrome",
			Address: common.HexToAddress("0x416b433906b1B72FA758e166e239c43d68dC6F29"),
			URL:     "https://velodrome.finance/liquidity",
		},
	},
	137: { // Polygon
		{
			Name:    "Uniswap",
			Address: common.HexToAddress("0xC36442b4a4522E871399CD717aBDD847Ab11FE88"),
			URL:     "https://app.uniswap.org/positions",
		},
		{
			Name:    "QuickSwap",
			Address: common.HexToAddress("0x8eF88E4c7CfbbaC1C163f7eddd4B578792201de6"),
			URL:     "https://quickswap.exchange/#/pools",
		},
		{
			Name:    "SushiSwap",
			Address: common.HexToAddress("0x2214A42d8e2A1d20635c2cb0664422c528B6A432"),
			URL:     "https://www.sushi.com/pool",
		},
	},
	56: { // BNB Smart Chain
		{
			Name:    "PancakeSwap",
			Address: common.HexToAddress("0x46A15B0b27311cedF172AB29E4f4766fbE7F4364"),
			URL:     "https://pancakeswap.finance/liquidity",
		},
		{
			Name:    "Uniswap",
			Address: common.HexToAddress("0x7b8A01B39D58278b5DE7e48c8449c9f4F5170613"),
			URL:     "https://app.uniswap.org/positions",
		},
		{
			Name:    "SushiSwap",
			Address: common.HexToAddress("0x2214A42d8e2A1d20635c2cb0664422c528B6A432"),
			URL:     "https://www.sushi.com/pool",
		},
	},
	8453: { // Base
		{
			Name:    "Uniswap",
			Address: common.HexToAddress("0x03a520b32C04BF3bEEf7BEb72E919cf822Ed34f1"),
			URL:     "https://app.uniswap.org/positions",
		},
		{
			Name:    "Aerodrome",
			Address: common.HexToAddress("0x827922686190790b37229fd06084350E74485b72"),
			URL:     "https://aerodrome.finance/liquidity",
		},
		{
			Name:    "SushiSwap",
			Address: common.HexToAddress("0x80C7DD17B01855a6D2347444a0FCC36136a314de"),
			URL:     "https://www.sushi.com/pool",
		},
		{
			Name:    "PancakeSwap",
			Address: common.HexToAddress("0x46A15B0b27311cedF172AB29E4f4766fbE7F4364"),
			URL:     "https://pancakeswap.finance/liquidity",
		},
	},
	43114: { // Avalanche
		{
			Name:    "Uniswap",
			Address: common.HexToAddress("0x655C406EBFa14EE2006250925e54ec43AD184f8B"),
			URL:     "https://app.uniswap.org/positions",
		},
		{
			Name:    "TraderJoe",
			Address: common.HexToAddress("0x7BFd7192E76D950832c77BB412aaE841049D8D9B"),
			URL:     "https://traderjoexyz.com/avalanche/pool",
		},
		{
			Name:    "SushiSwap",
			Address: common.HexToAddress("0x2214A42d8e2A1d20635c2cb0664422c528B6A432"),
			URL:     "https://www.sushi.com/pool",
		},
	},
	369: { // PulseChain
		{
			Name:    "9Inch",
			Address: common.HexToAddress("0x18A532b36A9F6B10b3FEC5BF225C00A0Ec89B79E"),
			URL:     "https://v3.9inch.io/?chain=pulse",
		},
		{
			Name:    "9mm",
			Address: common.HexToAddress("0xCC05bf158202b4F461Ede8843d76dcd7Bbad07f2"),
			URL:     "https://v3.9mm.pro",
		},
		{
			Name:    "PulseX",
			Address: common.HexToAddress("0x1a6c311a6d865dc3f35fea3e74d8c2f032ee7aa2"),
			URL:     "https://app.pulsex.com/liquidity",
		},
	},
	324: { // zkSync Era
		{
			Name:    "PancakeSwap",
			Address: common.HexToAddress("0xa815e2eD7f7d5B0c49fda367F249232a1B9D2883"),
			URL:     "https://pancakeswap.finance/liquidity",
		},
		{
			Name:    "SyncSwap",
			Address: common.HexToAddress("0x8f5E695569D47F86E3229B2b41cf8e64E5Ddb5b5"),
			URL:     "https://syncswap.xyz/pools",
		},
	},
	250: { // Fantom
		{
			Name:    "SpookySwap",
			Address: common.HexToAddress("0x89f9F823A234E71A3C92bb66cf7c2bc2e9B69092"),
			URL:     "https://spooky.fi/#/pools",
		},
		{
			Name:    "SushiSwap",
			Address: common.HexToAddress("0x2214A42d8e2A1d20635c2cb0664422c528B6A432"),
			URL:     "https://www.sushi.com/pool",
		},
	},
	59144: { // Linea
		{
			Name:    "PancakeSwap",
			Address: common.HexToAddress("0x46A15B0b27311cedF172AB29E4f4766fbE7F4364"),
			URL:     "https://pancakeswap.finance/liquidity",
		},
		{
			Name:    "Lynex",
			Address: common.HexToAddress("0xAB5ed8c81B65b6Bd2bC5F2DF4b59d81d6b6e5e9a"),
			URL:     "https://app.lynex.fi/liquidity",
		},
	},
	534352: { // Scroll
		{
			Name:    "Uniswap",
			Address: common.HexToAddress("0xB39002E4033b162fAc607fc3471E205FA2aE5967"),
			URL:     "https://app.uniswap.org/positions",
		},
		{
			Name:    "SushiSwap",
			Address: common.HexToAddress("0x80C7DD17B01855a6D2347444a0FCC36136a314de"),
			URL:     "https://www.sushi.com/pool",
		},
	},
	81457: { // Blast
		{
			Name:    "Thruster",
			Address: common.HexToAddress("0x434575EaEa081b735C985FA9bf63CD7b87e227F9"),
			URL:     "https://app.thruster.finance/liquidity",
		},
		{
			Name:    "SushiSwap",
			Address: common.HexToAddress("0x80C7DD17B01855a6D2347444a0FCC36136a314de"),
			URL:     "https://www.sushi.com/pool",
		},
	},
}
