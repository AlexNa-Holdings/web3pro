package eth

import (
	_ "embed"
	"encoding/json"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/rs/zerolog/log"
)

//go:embed ABI/ERC20.json
var ERC20_ABI_JSON []byte
var ERC20_ABI abi.ABI

//go:embed ABI/multicall2.json
var MULTICALL2_ABI_JSON []byte
var MULTICALL2_ABI abi.ABI

func LoadABIs() {
	err := json.Unmarshal(ERC20_ABI_JSON, &ERC20_ABI)
	if err != nil {
		log.Fatal().Msgf("Error unmarshaling ERC20 ABI: %v\n", err)
	}

	err = json.Unmarshal(MULTICALL2_ABI_JSON, &MULTICALL2_ABI)
	if err != nil {
		log.Fatal().Msgf("Error unmarshaling MULTICALL2 ABI: %v\n", err)
	}
}
