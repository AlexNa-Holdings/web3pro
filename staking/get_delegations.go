package staking

import (
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

// getDelegations queries the Monad staking precompile to find all validators
// an address has delegated to. Uses getDelegations(address,uint64) which returns
// paginated results.
func getDelegations(msg *bus.Message) (*bus.B_Staking_GetDelegations_Response, error) {
	req, ok := msg.Data.(*bus.B_Staking_GetDelegations)
	if !ok {
		return nil, fmt.Errorf("get_delegations: invalid data: %v", msg.Data)
	}

	var allValidatorIds []uint64
	var startValId uint64 = 0

	for {
		data, err := packGetDelegationsCall(req.Owner, startValId)
		if err != nil {
			log.Error().Err(err).Msg("Failed to pack getDelegations call")
			return nil, err
		}

		resp := bus.Fetch("eth", "call", &bus.B_EthCall{
			ChainId: req.ChainId,
			To:      req.Contract,
			From:    req.Owner,
			Data:    data,
		})

		if resp.Error != nil {
			log.Error().Err(resp.Error).Msg("eth call getDelegations")
			return nil, resp.Error
		}

		output, err := hexutil.Decode(resp.Data.(string))
		if err != nil {
			log.Error().Err(err).Msg("hexutil.Decode getDelegations")
			return nil, err
		}

		isDone, nextValId, valIds, err := unpackGetDelegationsResult(output)
		if err != nil {
			log.Error().Err(err).Msg("unpackGetDelegationsResult")
			return nil, err
		}

		log.Trace().
			Bool("isDone", isDone).
			Uint64("nextValId", nextValId).
			Int("count", len(valIds)).
			Msg("getDelegations page result")

		allValidatorIds = append(allValidatorIds, valIds...)

		if isDone || len(valIds) == 0 {
			break
		}
		startValId = nextValId
	}

	return &bus.B_Staking_GetDelegations_Response{
		ValidatorIds: allValidatorIds,
	}, nil
}

// packGetDelegationsCall creates calldata for getDelegations(address,uint64)
func packGetDelegationsCall(delegator common.Address, startValId uint64) ([]byte, error) {
	abiJSON := `[{"name":"getDelegations","type":"function","inputs":[{"name":"delegator","type":"address"},{"name":"startValId","type":"uint64"}],"outputs":[{"name":"isDone","type":"bool"},{"name":"nextValId","type":"uint64"},{"name":"valIds","type":"uint64[]"}]}]`

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	return parsedABI.Pack("getDelegations", delegator, startValId)
}

// unpackGetDelegationsResult decodes the response from getDelegations
// Returns: (isDone bool, nextValId uint64, valIds []uint64, error)
func unpackGetDelegationsResult(output []byte) (bool, uint64, []uint64, error) {
	abiJSON := `[{"name":"getDelegations","type":"function","inputs":[{"name":"delegator","type":"address"},{"name":"startValId","type":"uint64"}],"outputs":[{"name":"isDone","type":"bool"},{"name":"nextValId","type":"uint64"},{"name":"valIds","type":"uint64[]"}]}]`

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return false, 0, nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	results, err := parsedABI.Unpack("getDelegations", output)
	if err != nil {
		return false, 0, nil, fmt.Errorf("failed to unpack: %w", err)
	}

	if len(results) != 3 {
		return false, 0, nil, fmt.Errorf("unexpected result count: %d", len(results))
	}

	isDone, ok := results[0].(bool)
	if !ok {
		return false, 0, nil, fmt.Errorf("failed to cast isDone")
	}

	nextValId, ok := results[1].(uint64)
	if !ok {
		return false, 0, nil, fmt.Errorf("failed to cast nextValId")
	}

	valIdsRaw, ok := results[2].([]uint64)
	if !ok {
		return false, 0, nil, fmt.Errorf("failed to cast valIds")
	}

	return isDone, nextValId, valIdsRaw, nil
}
