package lp_v3

import (
	_ "embed"
	"encoding/json"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/ava-labs/coreth/accounts/abi"
	"github.com/rs/zerolog/log"
)

//go:embed ABI/NonfungiblePositionManager.json
var V3_MANAGER_JSON []byte
var V3_MANAGER abi.ABI

//go:embed ABI/v3Factory.json
var V3_FACTORY_JSON []byte
var V3_FACTORY abi.ABI

//go:embed ABI/v3Pool.json
var V3_POOL_JSON []byte
var V3_POOL abi.ABI

func Init() {
	err := json.Unmarshal(V3_MANAGER_JSON, &V3_MANAGER)
	if err != nil {
		log.Fatal().Msgf("Error unmarshaling V3_ABI_JSON ABI: %v\n", err)
	}

	err = json.Unmarshal(V3_FACTORY_JSON, &V3_FACTORY)
	if err != nil {
		log.Fatal().Msgf("Error unmarshaling V3_FACTORY_JSON ABI: %v\n", err)
	}

	err = json.Unmarshal(V3_POOL_JSON, &V3_POOL)
	if err != nil {
		log.Fatal().Msgf("Error unmarshaling V3_POOL_JSON ABI: %v\n", err)
	}

	go Loop()
}

func Loop() {
	ch := bus.Subscribe("lp_v3")
	for msg := range ch {
		if msg.RespondTo != 0 {
			continue // ignore responses
		}
		go process(msg)
	}
}

func process(msg *bus.Message) {
	switch msg.Topic {
	case "lp_v3":
		switch msg.Type {
		case "discover":
			err := discover(msg)
			msg.Respond(nil, err)
		case "get-nft-position":
			data, err := get_nft_position(msg)
			msg.Respond(data, err)
		case "get-factory":
			data, err := get_factory(msg)
			msg.Respond(data, err)
		case "get-pool":
			data, err := get_pool(msg)
			msg.Respond(data, err)
		case "get-pool-position":
			data, err := get_pool_position(msg)
			msg.Respond(data, err)

		case "get-price": //getSqrtPriceX96
			data, err := get_price(msg)
			msg.Respond(data, err)
		default:
			log.Error().Msgf("lp_v3: unknown type: %v", msg.Type)

		}
	}
}
