package eth

import (
	_ "embed"
	"encoding/json"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/rs/zerolog/log"
)

//go:embed ABI/ERC20.json
var ERC20_ABI_JSON []byte
var ERC20 abi.ABI

func LoadABIs() {
	err := json.Unmarshal(ERC20_ABI_JSON, &ERC20)
	if err != nil {
		log.Fatal().Msgf("Error unmarshaling ERC20 ABI: %v\n", err)
	}
}
