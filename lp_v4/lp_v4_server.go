package lp_v4

import (
	_ "embed"
	"encoding/json"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/rs/zerolog/log"
)

//go:embed ABI/PositionManager.json
var V4_POSITION_MANAGER_JSON []byte
var V4_POSITION_MANAGER abi.ABI

//go:embed ABI/PoolManager.json
var V4_POOL_MANAGER_JSON []byte
var V4_POOL_MANAGER abi.ABI

//go:embed ABI/StateView.json
var V4_STATE_VIEW_JSON []byte
var V4_STATE_VIEW abi.ABI

func Init() {
	err := json.Unmarshal(V4_POSITION_MANAGER_JSON, &V4_POSITION_MANAGER)
	if err != nil {
		log.Fatal().Msgf("Error unmarshaling V4_POSITION_MANAGER_JSON ABI: %v\n", err)
	}

	err = json.Unmarshal(V4_POOL_MANAGER_JSON, &V4_POOL_MANAGER)
	if err != nil {
		log.Fatal().Msgf("Error unmarshaling V4_POOL_MANAGER_JSON ABI: %v\n", err)
	}

	err = json.Unmarshal(V4_STATE_VIEW_JSON, &V4_STATE_VIEW)
	if err != nil {
		log.Fatal().Msgf("Error unmarshaling V4_STATE_VIEW_JSON ABI: %v\n", err)
	}

	go Loop()
}

func Loop() {
	ch := bus.Subscribe("lp_v4")
	for msg := range ch {
		if msg.RespondTo != 0 {
			continue // ignore responses
		}
		go process(msg)
	}
}

func process(msg *bus.Message) {
	switch msg.Topic {
	case "lp_v4":
		switch msg.Type {
		case "discover":
			err := discover(msg)
			msg.Respond(nil, err)
		case "get-position-status":
			data, err := getPositionStatus(msg)
			msg.Respond(data, err)
		case "get-nft-position":
			data, err := getNftPosition(msg)
			msg.Respond(data, err)
		default:
			log.Error().Msgf("lp_v4: unknown type: %v", msg.Type)
		}
	}
}
