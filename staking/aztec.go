package staking

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

// Aztec Token Vault ABI fragments
const aztecVaultABI = `[
	{"name":"getBeneficiary","type":"function","inputs":[],"outputs":[{"name":"","type":"address"}]},
	{"name":"getAllocation","type":"function","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"name":"getClaimable","type":"function","inputs":[],"outputs":[{"name":"","type":"uint256"}]},
	{"name":"getToken","type":"function","inputs":[],"outputs":[{"name":"","type":"address"}]}
]`

const erc20BalanceOfABI = `[{"name":"balanceOf","type":"function","inputs":[{"name":"account","type":"address"}],"outputs":[{"name":"","type":"uint256"}]}]`

// getAztecBalance returns total allocation and staked percentage for an Aztec Token Vault
// Balance = getAllocation() (total in vault)
// StakedPercent = (allocation - vaultBalance) / allocation * 100
func getAztecBalance(req *bus.B_Staking_GetBalance, staking *cmn.Staking) (*bus.B_Staking_GetBalance_Response, error) {
	// For Aztec, the VaultAddress must be provided in a StakingPosition
	// The req.Contract should be the vault address for hardcoded providers
	vaultAddress := req.Contract

	// If VaultAddress is empty, we can't proceed
	if vaultAddress == (common.Address{}) {
		return nil, fmt.Errorf("aztec: vault address not provided")
	}

	log.Trace().Str("vault", vaultAddress.Hex()).Str("owner", req.Owner.Hex()).Msg("aztec: getting balance")

	// Parse vault ABI
	parsedVaultABI, err := abi.JSON(strings.NewReader(aztecVaultABI))
	if err != nil {
		return nil, fmt.Errorf("aztec: failed to parse vault ABI: %w", err)
	}

	// Get allocation from vault
	allocationData, err := parsedVaultABI.Pack("getAllocation")
	if err != nil {
		return nil, fmt.Errorf("aztec: failed to pack getAllocation: %w", err)
	}

	allocationResp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: req.ChainId,
		To:      vaultAddress,
		Data:    allocationData,
	})
	if allocationResp.Error != nil {
		return nil, fmt.Errorf("aztec: getAllocation failed: %w", allocationResp.Error)
	}

	allocationOutput, err := hexutil.Decode(allocationResp.Data.(string))
	if err != nil {
		return nil, fmt.Errorf("aztec: failed to decode allocation: %w", err)
	}
	allocation := new(big.Int).SetBytes(allocationOutput)

	log.Trace().Str("vault", vaultAddress.Hex()).Str("allocation", allocation.String()).Msg("aztec: got allocation")

	// Get token address from vault
	tokenData, err := parsedVaultABI.Pack("getToken")
	if err != nil {
		return nil, fmt.Errorf("aztec: failed to pack getToken: %w", err)
	}

	tokenResp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: req.ChainId,
		To:      vaultAddress,
		Data:    tokenData,
	})
	if tokenResp.Error != nil {
		return nil, fmt.Errorf("aztec: getToken failed: %w", tokenResp.Error)
	}

	tokenOutput, err := hexutil.Decode(tokenResp.Data.(string))
	if err != nil {
		return nil, fmt.Errorf("aztec: failed to decode token: %w", err)
	}
	tokenAddress := common.BytesToAddress(tokenOutput[12:32]) // Extract address from 32 bytes

	log.Trace().Str("vault", vaultAddress.Hex()).Str("token", tokenAddress.Hex()).Msg("aztec: got token address")

	// Get vault's token balance (unstaked amount)
	parsedERC20ABI, err := abi.JSON(strings.NewReader(erc20BalanceOfABI))
	if err != nil {
		return nil, fmt.Errorf("aztec: failed to parse ERC20 ABI: %w", err)
	}

	balanceData, err := parsedERC20ABI.Pack("balanceOf", vaultAddress)
	if err != nil {
		return nil, fmt.Errorf("aztec: failed to pack balanceOf: %w", err)
	}

	balanceResp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: req.ChainId,
		To:      tokenAddress,
		Data:    balanceData,
	})
	if balanceResp.Error != nil {
		return nil, fmt.Errorf("aztec: balanceOf failed: %w", balanceResp.Error)
	}

	balanceOutput, err := hexutil.Decode(balanceResp.Data.(string))
	if err != nil {
		return nil, fmt.Errorf("aztec: failed to decode balance: %w", err)
	}
	unstakedBalance := new(big.Int).SetBytes(balanceOutput)

	log.Trace().Str("vault", vaultAddress.Hex()).Str("unstakedBalance", unstakedBalance.String()).Msg("aztec: got vault token balance")

	// Staked = Allocation - Unstaked Balance
	stakedBalance := new(big.Int).Sub(allocation, unstakedBalance)
	if stakedBalance.Sign() < 0 {
		stakedBalance = big.NewInt(0)
	}

	// Calculate staked percentage
	var stakedPercent float64
	if allocation.Sign() > 0 {
		// stakedPercent = staked / allocation * 100
		stakedFloat := new(big.Float).SetInt(stakedBalance)
		allocationFloat := new(big.Float).SetInt(allocation)
		percentFloat := new(big.Float).Quo(stakedFloat, allocationFloat)
		percentFloat.Mul(percentFloat, big.NewFloat(100))
		stakedPercent, _ = percentFloat.Float64()
	}

	log.Trace().Str("vault", vaultAddress.Hex()).Str("allocation", allocation.String()).Str("staked", stakedBalance.String()).Float64("stakedPercent", stakedPercent).Msg("aztec: calculated balance")

	return &bus.B_Staking_GetBalance_Response{
		Balance:       allocation, // Return total allocation as balance
		StakedPercent: stakedPercent,
	}, nil
}

// getAztecPending gets claimable rewards for an Aztec Token Vault
func getAztecPending(req *bus.B_Staking_GetPending, staking *cmn.Staking) (*bus.B_Staking_GetPending_Response, error) {
	// For Aztec, the VaultAddress must be provided
	vaultAddress := req.Contract

	if vaultAddress == (common.Address{}) {
		return nil, fmt.Errorf("aztec: vault address not provided")
	}

	log.Trace().Str("vault", vaultAddress.Hex()).Msg("aztec: getting pending rewards")

	// Parse vault ABI
	parsedVaultABI, err := abi.JSON(strings.NewReader(aztecVaultABI))
	if err != nil {
		return nil, fmt.Errorf("aztec: failed to parse vault ABI: %w", err)
	}

	// Get claimable rewards
	claimableData, err := parsedVaultABI.Pack("getClaimable")
	if err != nil {
		return nil, fmt.Errorf("aztec: failed to pack getClaimable: %w", err)
	}

	claimableResp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: req.ChainId,
		To:      vaultAddress,
		Data:    claimableData,
	})
	if claimableResp.Error != nil {
		return nil, fmt.Errorf("aztec: getClaimable failed: %w", claimableResp.Error)
	}

	claimableOutput, err := hexutil.Decode(claimableResp.Data.(string))
	if err != nil {
		return nil, fmt.Errorf("aztec: failed to decode claimable: %w", err)
	}
	claimable := new(big.Int).SetBytes(claimableOutput)

	log.Trace().Str("vault", vaultAddress.Hex()).Str("claimable", claimable.String()).Msg("aztec: got claimable rewards")

	return &bus.B_Staking_GetPending_Response{
		Pending: claimable,
	}, nil
}

// ValidateAztecVault validates that a vault address is valid and returns the beneficiary
func ValidateAztecVault(chainId int, vaultAddress common.Address) (common.Address, error) {
	// Parse vault ABI
	parsedVaultABI, err := abi.JSON(strings.NewReader(aztecVaultABI))
	if err != nil {
		return common.Address{}, fmt.Errorf("aztec: failed to parse vault ABI: %w", err)
	}

	// Get beneficiary
	beneficiaryData, err := parsedVaultABI.Pack("getBeneficiary")
	if err != nil {
		return common.Address{}, fmt.Errorf("aztec: failed to pack getBeneficiary: %w", err)
	}

	beneficiaryResp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: chainId,
		To:      vaultAddress,
		Data:    beneficiaryData,
	})
	if beneficiaryResp.Error != nil {
		return common.Address{}, fmt.Errorf("aztec: getBeneficiary failed - invalid vault?: %w", beneficiaryResp.Error)
	}

	beneficiaryOutput, err := hexutil.Decode(beneficiaryResp.Data.(string))
	if err != nil {
		return common.Address{}, fmt.Errorf("aztec: failed to decode beneficiary: %w", err)
	}

	beneficiary := common.BytesToAddress(beneficiaryOutput[12:32])

	log.Trace().Str("vault", vaultAddress.Hex()).Str("beneficiary", beneficiary.Hex()).Msg("aztec: validated vault")

	return beneficiary, nil
}
