package cmn

import "github.com/ethereum/go-ethereum/common"

type PD_V2 struct {
	Name       string
	Factory    common.Address
	Router     common.Address
	URL        string
	SubgraphID string // The Graph subgraph ID or full URL (if starts with http)
}

// SubgraphID can be either:
// - A subgraph ID (combined with Config.TheGraphGateway at runtime)
// - A full URL starting with "http" (used as-is, e.g., for PulseChain's own Graph)
//
// Get your API key at https://thegraph.com/studio/apikeys/
//
// NOTE: Only subgraphs with "liquidityPositions" entity support discovery.
// PancakeSwap and some other DEXes use a different schema without user position tracking.
var PrefedinedLP_V2 = map[int]([]PD_V2){
	1: { // Ethereum Mainnet
		{
			Name:       "Uniswap",
			Factory:    common.HexToAddress("0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f"),
			Router:     common.HexToAddress("0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"),
			URL:        "https://app.uniswap.org/positions",
			SubgraphID: "A3Np3RQbaBA6oKJgiwDJeo5T3zrYfGHPWFYayMwtNDum",
		},
		{
			Name:       "SushiSwap",
			Factory:    common.HexToAddress("0xC0AEe478e3658e2610c5F7A4A2E1777cE9e4f2Ac"),
			Router:     common.HexToAddress("0xd9e1cE17f2641f24aE83637ab66a2cca9C378B9F"),
			URL:        "https://www.sushi.com/pool",
			SubgraphID: "6NUtT5mGjZ1tSshKLf5Q3uEEJtjBZJo1TpL5MXsUBqrT",
		},
	},
	42161: { // Arbitrum One
		{
			Name:       "Uniswap",
			Factory:    common.HexToAddress("0xf1D7CC64Fb4452F05c498126312eBE29f30Fbcf9"),
			Router:     common.HexToAddress("0x4752ba5dbc23f44d87826276bf6fd6b1c372ad24"),
			URL:        "https://app.uniswap.org/positions",
			SubgraphID: "EYCKATKGBKLWvSfwvBjzfCBmGwYNdVkduYXVivCsLRFu",
		},
		{
			Name:       "SushiSwap",
			Factory:    common.HexToAddress("0xc35DADB65012eC5796536bD9864eD8773aBc74C4"),
			Router:     common.HexToAddress("0x1b02dA8Cb0d097eB8D57A175b88c7D8b47997506"),
			URL:        "https://www.sushi.com/pool",
			SubgraphID: "8nFDCAhdnJQEhQF3ZRnfWkJ6FkRsfAiiVabVn4eGoAZH",
		},
		{
			Name:       "Camelot",
			Factory:    common.HexToAddress("0x6EcCab422D763aC031210895C81787E87B43A652"),
			Router:     common.HexToAddress("0xc873fEcbd354f5A56E00E710B90EF4201db2448d"),
			URL:        "https://app.camelot.exchange/liquidity",
			SubgraphID: "", // Camelot subgraph doesn't have liquidityPositions entity
		},
	},
	10: { // Optimism
		{
			Name:       "Uniswap",
			Factory:    common.HexToAddress("0x0c3c1c532F1e39EdF36BE9Fe0bE1410313E074Bf"),
			Router:     common.HexToAddress("0x4A7b5Da61326A6379179b40d00F57E5bbDC962c2"),
			URL:        "https://app.uniswap.org/positions",
			SubgraphID: "", // No V2 subgraph found for Optimism
		},
		{
			Name:       "Velodrome",
			Factory:    common.HexToAddress("0x25CbdDb98b35ab1FF77413456B31EC81A6B6B746"),
			Router:     common.HexToAddress("0x9c12939390052919aF3155f41Bf4160Fd3666A6f"),
			URL:        "https://velodrome.finance/liquidity",
			SubgraphID: "", // Velodrome subgraph doesn't have liquidityPositions entity
		},
	},
	137: { // Polygon
		{
			Name:       "Uniswap",
			Factory:    common.HexToAddress("0x9e5A52f57b3038F1B8EeE45F28b3C1967e22799C"),
			Router:     common.HexToAddress("0xedf6066a2b290C185783862C7F4776A2C8077AD1"),
			URL:        "https://app.uniswap.org/positions",
			SubgraphID: "",
		},
		{
			Name:       "QuickSwap",
			Factory:    common.HexToAddress("0x5757371414417b8C6CAad45bAeF941aBc7d3Ab32"),
			Router:     common.HexToAddress("0xa5E0829CaCEd8fFDD4De3c43696c57F7D7A678ff"),
			URL:        "https://quickswap.exchange/#/pools",
			SubgraphID: "FqsRcH1XqSjqVx9GRTvEJe959aCbKrcyGgDWBrUkG24g",
		},
		{
			Name:       "SushiSwap",
			Factory:    common.HexToAddress("0xc35DADB65012eC5796536bD9864eD8773aBc74C4"),
			Router:     common.HexToAddress("0x1b02dA8Cb0d097eB8D57A175b88c7D8b47997506"),
			URL:        "https://www.sushi.com/pool",
			SubgraphID: "8obLTNcEuGMqkTMFpoNnqNzwRzNvM7o7L1rHo2RDkPCN",
		},
	},
	56: { // BNB Smart Chain
		{
			Name:       "PancakeSwap",
			Factory:    common.HexToAddress("0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73"),
			Router:     common.HexToAddress("0x10ED43C718714eb63d5aA57B78B54704E256024E"),
			URL:        "https://pancakeswap.finance/liquidity",
			SubgraphID: "EsL7geTRcA3LaLLM9EcMFzYbUgnvf8RixoEEGErrodB3",
		},
		{
			Name:       "SushiSwap",
			Factory:    common.HexToAddress("0xc35DADB65012eC5796536bD9864eD8773aBc74C4"),
			Router:     common.HexToAddress("0x1b02dA8Cb0d097eB8D57A175b88c7D8b47997506"),
			URL:        "https://www.sushi.com/pool",
			SubgraphID: "",
		},
	},
	8453: { // Base
		{
			Name:       "Uniswap",
			Factory:    common.HexToAddress("0x8909Dc15e40173Ff4699343b6eB8132c65e18eC6"),
			Router:     common.HexToAddress("0x4752ba5dbc23f44d87826276bf6fd6b1c372ad24"),
			URL:        "https://app.uniswap.org/positions",
			SubgraphID: "", // Subgraph doesn't have liquidityPositions entity
		},
		{
			Name:       "PancakeSwap",
			Factory:    common.HexToAddress("0x02a84c1b3BBD7401a5f7fa98a384EBC70bB5749E"),
			Router:     common.HexToAddress("0x8cFe327CEc66d1C090Dd72bd0FF11d690C33a2Eb"),
			URL:        "https://pancakeswap.finance/liquidity",
			SubgraphID: "2NjL7L4CmQaGJSacM43ofmH6ARf6gJoBeBaJtz9eWAQ9",
		},
		{
			Name:       "Aerodrome",
			Factory:    common.HexToAddress("0x420DD381b31aEf6683db6B902084cB0FFECe40Da"),
			Router:     common.HexToAddress("0xcF77a3Ba9A5CA399B7c97c74d54e5b1Beb874E43"),
			URL:        "https://aerodrome.finance/liquidity",
			SubgraphID: "", // Aerodrome subgraph doesn't have liquidityPositions entity
		},
		{
			Name:       "SushiSwap",
			Factory:    common.HexToAddress("0x71524B4f93c58fcbF659783284E38825f0622859"),
			Router:     common.HexToAddress("0x6BDED42c6DA8FBf0d2bA55B2fa120C5e0c8D7891"),
			URL:        "https://www.sushi.com/pool",
			SubgraphID: "", // No V2 subgraph found for Base
		},
	},
	43114: { // Avalanche
		{
			Name:       "TraderJoe",
			Factory:    common.HexToAddress("0x9Ad6C38BE94206cA50bb0d90783181c47F58c0a8"),
			Router:     common.HexToAddress("0x60aE616a2155Ee3d9A68541Ba4544862310933d4"),
			URL:        "https://traderjoexyz.com/avalanche/pool",
			SubgraphID: "5Mw8qH6fRMCRxYXee7NqLh9FP5i5TMT5GFjFaHZMgxkF",
		},
		{
			Name:       "SushiSwap",
			Factory:    common.HexToAddress("0xc35DADB65012eC5796536bD9864eD8773aBc74C4"),
			Router:     common.HexToAddress("0x1b02dA8Cb0d097eB8D57A175b88c7D8b47997506"),
			URL:        "https://www.sushi.com/pool",
			SubgraphID: "6nDfs3qv13SvhCr1PUD8M6hrWQ88xEQAvMGJNyucC4aq",
		},
	},
	369: { // PulseChain
		{
			Name:       "PulseX V1",
			Factory:    common.HexToAddress("0x1715a3E4A142d8b698131108995174F37aEBA10D"),
			Router:     common.HexToAddress("0x98bf93ebf5c380C0e6Ae8e192A7e2AE08edAcc02"),
			URL:        "https://app.pulsex.com/liquidity",
			SubgraphID: "https://graph.pulsechain.com/subgraphs/name/pulsechain/pulsex",
		},
		{
			Name:       "PulseX V2",
			Factory:    common.HexToAddress("0x29eA7545DEf87022BAdc76323F373EA1e707C523"),
			Router:     common.HexToAddress("0x165C3410fC91EF562C50559f7d2289fEbed552d9"),
			URL:        "https://app.pulsex.com/liquidity",
			SubgraphID: "https://graph.pulsechain.com/subgraphs/name/pulsechain/pulsexv2",
		},
		{
			Name:       "9mm V2",
			Factory:    common.HexToAddress("0x3a0Fa7884dD93f3cd234bBE2A0958Ef04b05E13b"),
			Router:     common.HexToAddress("0xcC73b59F8D6e5b0DE5bBf5eA186e5F0C888b4208"),
			URL:        "https://swap.9mm.pro",
			SubgraphID: "https://info-api.9mm.pro/subgraphs/name/pulsechain/9mm",
		},
	},
	250: { // Fantom
		{
			Name:       "SpookySwap",
			Factory:    common.HexToAddress("0x152eE697f2E276fA89E96742e9bB9aB1F2E61bE3"),
			Router:     common.HexToAddress("0xF491e7B69E4244ad4002BC14e878a34207E38c29"),
			URL:        "https://spooky.fi/#/pools",
			SubgraphID: "2VaKGu3kDewSd3mBmhAJcBA1snJnWPnS3v8wpACz3tLM",
		},
		{
			Name:       "SushiSwap",
			Factory:    common.HexToAddress("0xc35DADB65012eC5796536bD9864eD8773aBc74C4"),
			Router:     common.HexToAddress("0x1b02dA8Cb0d097eB8D57A175b88c7D8b47997506"),
			URL:        "https://www.sushi.com/pool",
			SubgraphID: "3nozHgmPEDLRSgLPKfVD9XKdrajkumuhfE4Cb8Q6GLMK",
		},
	},
}
