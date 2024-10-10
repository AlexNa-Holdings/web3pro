package cmn

import "github.com/ethereum/go-ethereum/common"

type PD_V3 struct {
	Name    string
	Address common.Address
}

var PrefedinedLP_V3 = map[int]([]LP_V3){
	1: { // Ethereum
		{
			Name:    "Uniswap",
			Address: common.HexToAddress("0xC36442b4a4522E871399CD717aBDD847Ab11FE88"),
		},
	},
	369: { // PulseChain
		{
			Name:    "9Inch",
			Address: common.HexToAddress("0x18A532b36A9F6B10b3FEC5BF225C00A0Ec89B79E"),
		},
	},
}
