package cmn

import "github.com/ethereum/go-ethereum/common"

type PD_V3 struct {
	Name    string
	Address common.Address
	URL     string
}

var PrefedinedLP_V3 = map[int]([]PD_V3){
	1: { // Ethereum
		{
			Name:    "Uni",
			Address: common.HexToAddress("0xC36442b4a4522E871399CD717aBDD847Ab11FE88"),
			URL:     "https://app.uniswap.org/swap",
		},
		{
			Name:    "Pancake",
			Address: common.HexToAddress("0x46A15B0b27311cedF172AB29E4f4766fbE7F4364"),
			URL:     "https://pancakeswap.finance/swap&chin=eth",
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
	},
	56: { // Binance Smart Chain
		{
			Name:    "Pancake",
			Address: common.HexToAddress("0x46A15B0b27311cedF172AB29E4f4766fbE7F4364"),
			URL:     "https://pancakeswap.finance/swap&chin=eth",
		},
	},
}
