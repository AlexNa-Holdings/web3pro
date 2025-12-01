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

func getBalance(msg *bus.Message) (*bus.B_Staking_GetBalance_Response, error) {
	req, ok := msg.Data.(*bus.B_Staking_GetBalance)
	if !ok {
		return nil, fmt.Errorf("get_balance: invalid data: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("get_balance: no wallet")
	}

	// Find the staking config
	staking := w.GetStaking(req.ChainId, req.Contract)
	if staking == nil {
		return nil, fmt.Errorf("get_balance: staking not found for chain %d contract %s", req.ChainId, req.Contract.Hex())
	}

	// Build the function call based on the balance function name
	funcName := staking.BalanceFunc
	if funcName == "" {
		funcName = "balanceOf"
	}

	// Use request's ValidatorId if provided, otherwise use staking provider's
	validatorId := req.ValidatorId
	if validatorId == 0 {
		validatorId = staking.ValidatorId
	}

	var data []byte
	var err error

	// Check if this is validator-based staking (e.g., Monad native staking)
	if validatorId > 0 {
		// Use getDelegator(uint64,address) for validator staking
		data, err = packValidatorCall(funcName, validatorId, req.Owner)
	} else {
		// Create a dynamic ABI for the function
		// Most staking contracts use balanceOf(address) -> uint256
		data, err = packCall(funcName, req.Owner)
	}
	if err != nil {
		log.Error().Err(err).Msgf("Failed to pack %s call", funcName)
		return nil, err
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: req.ChainId,
		To:      req.Contract,
		From:    req.Owner,
		Data:    data,
	})

	if resp.Error != nil {
		log.Error().Err(resp.Error).Msgf("eth call %s", funcName)
		return nil, resp.Error
	}

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		log.Error().Err(err).Msgf("hexutil.Decode %s", funcName)
		return nil, err
	}

	var balance *big.Int

	// For validator staking, getDelegator returns a struct - extract the staked amount (first field)
	if validatorId > 0 && len(output) >= 32 {
		balance = new(big.Int).SetBytes(output[:32])
	} else {
		// Decode the uint256 result
		balance = new(big.Int).SetBytes(output)
	}

	return &bus.B_Staking_GetBalance_Response{
		Balance: balance,
	}, nil
}

// packCall creates calldata for a function that takes an address and returns uint256
func packCall(funcName string, addr common.Address) ([]byte, error) {
	// Create a simple ABI for function(address) returns (uint256)
	abiJSON := fmt.Sprintf(`[{"name":"%s","type":"function","inputs":[{"name":"account","type":"address"}],"outputs":[{"name":"","type":"uint256"}]}]`, funcName)

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	return parsedABI.Pack(funcName, addr)
}

// packValidatorCall creates calldata for a function that takes (uint64, address) for validator-based staking
// Used for native staking precompiles like Monad's getDelegator(uint64,address)
func packValidatorCall(funcName string, validatorId uint64, addr common.Address) ([]byte, error) {
	abiJSON := fmt.Sprintf(`[{"name":"%s","type":"function","inputs":[{"name":"validatorId","type":"uint64"},{"name":"delegator","type":"address"}],"outputs":[{"name":"","type":"uint256"}]}]`, funcName)

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	return parsedABI.Pack(funcName, validatorId, addr)
}
