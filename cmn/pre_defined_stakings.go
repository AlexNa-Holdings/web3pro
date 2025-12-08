package cmn

import "github.com/ethereum/go-ethereum/common"

var PredefinedStakings []Staking = []Staking{
	{
		Name:         "SAVVA Staking",
		ChainId:      369, // PulseChain
		Contract:     common.HexToAddress("0x6BC07cC3d6c0927320d273D6318ef5405f2eB911"),
		URL:          "https://savva.app",
		StakedToken:  common.HexToAddress("0xb528a9DB27A74dB802C74D0CCc40657efE5F0A45"),
		BalanceFunc:  "balanceOf",
		Reward1Token: common.HexToAddress("0xb528a9DB27A74dB802C74D0CCc40657efE5F0A45"),
		Reward1Func:  "claimable",
	},
	{
		Name:         "Liquid Loans",
		ChainId:      369, // PulseChain
		Contract:     common.HexToAddress("0x853F0CD4B0083eDf7cFf5Ad9A296f02Ffb71C995"),
		URL:          "https://go.liquidloans.io/#/staking-pool",
		StakedToken:  common.HexToAddress("0x9159f1D2a9f51998Fc9Ab03fbd8f265ab14A1b3B"), // LOAN
		BalanceFunc:  "stakes",
		Reward1Token: common.HexToAddress("0x0dEEd1486bc52aA0d3E6f8849cEC5adD6598A162"), // USDL
		Reward1Func:  "getPendingUSDLGain",
		Reward2Token: common.HexToAddress("0xA1077a294dDE1B09bB078844df40758a5D0f9a27"), // WPLS
		Reward2Func:  "getPendingPLSGain",
	},
	{
		Name:         "INC Printer",
		ChainId:      369, // PulseChain
		Contract:     common.HexToAddress("0x35b99f29b3Ec3276A2b3Bb5863326B1c100aa160"),
		URL:          "https://incprinter.com/#/",
		StakedToken:  common.HexToAddress("0x6c203a555824ec90a215f37916cf8db58ebe2fa3"), // PRINT
		BalanceFunc:  "stakes",
		Reward1Token: common.HexToAddress("0x144cd22aaa2a80fed0bb8b1deaddc51a53df1d50"), // INCD
		Reward1Func:  "getPendingLUSDGain",
		Reward2Token: common.HexToAddress("0xA1077a294dDE1B09bB078844df40758a5D0f9a27"), // WPLS
		Reward2Func:  "getPendingETHGain",
	},
	{
		Name:         "INC INCD-DAI",
		ChainId:      369, // PulseChain
		Contract:     common.HexToAddress("0x5A0D3cC13A523Dd7A9279C5Eb4f363593dA4198e"),
		URL:          "https://incprinter.com/#/",
		StakedToken:  common.HexToAddress("0x2cb92b1e8b2fc53b5a9165e765488e17b38c26d3"), // INCD-DAI LP
		BalanceFunc:  "balanceOf",
		Reward1Token: common.HexToAddress("0x6c203a555824ec90a215f37916cf8db58ebe2fa3"), // PRINT
		Reward1Func:  "earned",
	},
	{
		Name:         "INC PRINT-INC",
		ChainId:      369, // PulseChain
		Contract:     common.HexToAddress("0x857ab0cb7449Fb29429FC30596F08cfbf9F171F5"),
		URL:          "https://incprinter.com/#/",
		StakedToken:  common.HexToAddress("0xf35f8db9b6760799db76796340aacc69dea0c644"), // PRINT-INC LP
		BalanceFunc:  "balanceOf",
		Reward1Token: common.HexToAddress("0x6c203a555824ec90a215f37916cf8db58ebe2fa3"), // PRINT
		Reward1Func:  "earned",
	},
	// Monad Native Staking - ValidatorId must be set after adding (use staking edit)
	// Find your validator ID at https://monadvision.com/validators
	{
		Name:         "Monad Staking",
		ChainId:      143, // Monad
		Contract:     common.HexToAddress("0x0000000000000000000000000000000000001000"), // Staking precompile
		URL:          "https://monadvision.com/myspace?feature=Stake",
		StakedToken:  common.HexToAddress("0x0000000000000000000000000000000000000000"), // Native MON
		BalanceFunc:  "getDelegator",
		Reward1Token: common.HexToAddress("0x0000000000000000000000000000000000000000"), // Native MON rewards
		Reward1Func:  "getDelegator",
		ValidatorId:  0, // Set your validator ID after adding
	},
	// Aztec Staking - Hardcoded provider with vault-based positions
	// User must add their Token Vault address to create a position
	// Staked = getAllocation() - token.balanceOf(vault), Rewards = getClaimable()
	{
		Name:         "Aztec Staking",
		ChainId:      1, // Ethereum Mainnet
		Contract:     common.HexToAddress("0xa92ecFD0E70c9cd5E5cd76c50Af0F7Da93567a4f"), // GSE contract (for reference)
		URL:          "https://stake.aztec.network",
		StakedToken:  common.HexToAddress("0xA27EC0006e59f245217Ff08CD52A7E8b169E62D2"), // AZTEC token
		Reward1Token: common.HexToAddress("0xA27EC0006e59f245217Ff08CD52A7E8b169E62D2"), // AZTEC token (rewards)
		Hardcoded:    true,
	},
}
