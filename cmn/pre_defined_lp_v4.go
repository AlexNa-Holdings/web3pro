package cmn

import "github.com/ethereum/go-ethereum/common"

type PD_V4 struct {
	Name            string
	ProviderAddress common.Address // PositionManager
	PoolManager     common.Address // Singleton PoolManager
	StateView       common.Address // StateView contract for reading pool state
	URL             string
	SubgraphURL     string
}

var PrefedinedLP_V4 = map[int]([]PD_V4){
	1: { // Ethereum Mainnet
		{
			Name:            "Uniswap",
			ProviderAddress: common.HexToAddress("0xbD216513d74C8cf14cf4747E6AaA6420FF64ee9e"),
			PoolManager:     common.HexToAddress("0x000000000004444c5dc75cb358380d2e3de08a90"),
			StateView:       common.HexToAddress("0x7ffe42c4a5deea5b0fec41c94c136cf115597227"),
			URL:             "https://app.uniswap.org/positions",
			SubgraphURL:     "https://gateway.thegraph.com/api/{api-key}/subgraphs/id/DiYPVdygkfjDWhbxGSqAQxwBKmfKnkWQojqeM2rkLb3G",
		},
	},
	42161: { // Arbitrum One
		{
			Name:            "Uniswap",
			ProviderAddress: common.HexToAddress("0xd88F38F930b7952f2DB2432Cb002E7abbF3dD869"),
			PoolManager:     common.HexToAddress("0x360E68faCcca8cA495c1B759Fd9EEe466db9FB32"),
			StateView:       common.HexToAddress("0x76fd297e2d437cd7f76d50f01afe6160f86e9990"),
			URL:             "https://app.uniswap.org/positions",
			SubgraphURL:     "https://gateway.thegraph.com/api/{api-key}/subgraphs/id/GUWTydadTY23DvLudPv7MRhQkooMmSjpJ1tvcr9Lgkgq",
		},
	},
	10: { // Optimism
		{
			Name:            "Uniswap",
			ProviderAddress: common.HexToAddress("0x3C3Ea4B57a46241e54610e5f022E5c45859A1017"),
			PoolManager:     common.HexToAddress("0x9a13F98Cb987694C9F086b1F5eB990EeA8264Ec3"),
			StateView:       common.HexToAddress("0xc18a3169788f4f75a170290584eca6395c75ecdb"),
			URL:             "https://app.uniswap.org/positions",
			SubgraphURL:     "https://gateway.thegraph.com/api/{api-key}/subgraphs/id/5pSBzyD1GBFmqSBjLrDGMWPVYwEv2BNswsGvLxWEWNfR",
		},
	},
	137: { // Polygon
		{
			Name:            "Uniswap",
			ProviderAddress: common.HexToAddress("0x1Ec2eBf4F37E7363FDfe3f0E37A14CE0Ff938447"),
			PoolManager:     common.HexToAddress("0x67366782805870060151383F4BbFF9daB53e5cD6"),
			StateView:       common.HexToAddress("0x5ea1bd7974c8a611cbab0bdcafcb1d9cc9b3ba5a"),
			URL:             "https://app.uniswap.org/positions",
			SubgraphURL:     "https://gateway.thegraph.com/api/{api-key}/subgraphs/id/6NMLgPeuGdWk9pxHEwnUJK2xrMVuGYFWm2i4eWMSDGWC",
		},
	},
	56: { // BNB Smart Chain
		{
			Name:            "Uniswap",
			ProviderAddress: common.HexToAddress("0x7A4a5c919aE2541AeD11041A1AEeE68f1287f95b"),
			PoolManager:     common.HexToAddress("0x28e2Ea090877bF75740558f6BFB36A5ffeE9e9dF"),
			StateView:       common.HexToAddress("0xd13dd3d6e93f276fafc9db9e6bb47c1180aee0c4"),
			URL:             "https://app.uniswap.org/positions",
			SubgraphURL:     "https://gateway.thegraph.com/api/{api-key}/subgraphs/id/3HfqCn5cpVG5FLQ6JpVhDvnSJPqRu95KCJwqsKMgAt7X",
		},
	},
	8453: { // Base
		{
			Name:            "Uniswap",
			ProviderAddress: common.HexToAddress("0x7C5f5A4bBd8fD63184577525326123B519429bDc"),
			PoolManager:     common.HexToAddress("0x498581fF718922c3f8e6A244956aF099B2652b2b"),
			StateView:       common.HexToAddress("0xa3c0c9b65bad0b08107aa264b0f3db444b867a71"),
			URL:             "https://app.uniswap.org/positions",
			SubgraphURL:     "https://gateway.thegraph.com/api/{api-key}/subgraphs/id/GCZR3WReDkPmZSqkPWupSNTsssZSJqvHmCApPc4GqsxU",
		},
	},
	43114: { // Avalanche
		{
			Name:            "Uniswap",
			ProviderAddress: common.HexToAddress("0xb3CD9b4CF08b107ae6f3F6C9fD7Fb7bb7c2fA0E6"),
			PoolManager:     common.HexToAddress("0x06380C0e0912312B5150364B9DC4542BA0DbBc85"),
			StateView:       common.HexToAddress("0xc3c9e198c735a4b97e3e683f391ccbdd60b69286"),
			URL:             "https://app.uniswap.org/positions",
			SubgraphURL:     "https://gateway.thegraph.com/api/{api-key}/subgraphs/id/6GeeX9tBAquHtSm4So2pXXNmjuQJbm3HiLLsE6dKi4UD",
		},
	},
	324: { // zkSync Era
		{
			Name:            "Uniswap",
			ProviderAddress: common.HexToAddress("0x9a13F98Cb987694C9F086b1F5eB990EeA8264Ec3"),
			PoolManager:     common.HexToAddress("0x9a89c8E57D0fC74E55926289120B7B8A248D99f7"),
			StateView:       common.HexToAddress("0x0000000000000000000000000000000000000000"), // Not available
			URL:             "https://app.uniswap.org/positions",
			SubgraphURL:     "https://gateway.thegraph.com/api/{api-key}/subgraphs/id/4HRo4yiNyAWr1p3r6QfLB4K14cMDbR4JjmGUU5g4Y6xK",
		},
	},
	81457: { // Blast
		{
			Name:            "Uniswap",
			ProviderAddress: common.HexToAddress("0x4eE8DDd4f5697E5a7805Bebd5e4Cc118Cae9A903"),
			PoolManager:     common.HexToAddress("0x1631559198A9e474033433b2958daBC135ab6446"),
			StateView:       common.HexToAddress("0x12a88ae16f46dce4e8b15368008ab3380885df30"),
			URL:             "https://app.uniswap.org/positions",
			SubgraphURL:     "https://gateway.thegraph.com/api/{api-key}/subgraphs/id/Bc94BthHCL8qKsxfPcAuFGDMBwB6f5NcXkLhNq8sLJar",
		},
	},
	7777777: { // Zora
		{
			Name:            "Uniswap",
			ProviderAddress: common.HexToAddress("0x4eE8DDd4f5697E5a7805Bebd5e4Cc118Cae9A903"),
			PoolManager:     common.HexToAddress("0x0575338e4C17006aE181B47900A84404247CA30f"),
			StateView:       common.HexToAddress("0x385785af07d63b50d0a0ea57c4ff89d06adf7328"),
			URL:             "https://app.uniswap.org/positions",
			SubgraphURL:     "https://gateway.thegraph.com/api/{api-key}/subgraphs/id/BcrwnLuDhD7ny2BmWxLUGvEf1H4SgSBMEkPm5HJm9wMU",
		},
	},
	480: { // World Chain
		{
			Name:            "Uniswap",
			ProviderAddress: common.HexToAddress("0x4eE8DDd4f5697E5a7805Bebd5e4Cc118Cae9A903"),
			PoolManager:     common.HexToAddress("0xb1860D529182ac3BC1F51Fa2ABd56662b7D13f33"),
			StateView:       common.HexToAddress("0x51d394718bc09297262e368c1a481217fdeb71eb"),
			URL:             "https://app.uniswap.org/positions",
			SubgraphURL:     "https://gateway.thegraph.com/api/{api-key}/subgraphs/id/28xHHc4JPBrWRFxAdx1vAAXivUBuPdnUisCb8YjLBmvK",
		},
	},
}
