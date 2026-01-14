package lp_v2

import (
	_ "embed"
	"encoding/json"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/rs/zerolog/log"
)

//go:embed ABI/v2Factory.json
var V2_FACTORY_JSON []byte
var V2_FACTORY abi.ABI

//go:embed ABI/v2Pair.json
var V2_PAIR_JSON []byte
var V2_PAIR abi.ABI

func Init() {
	err := json.Unmarshal(V2_FACTORY_JSON, &V2_FACTORY)
	if err != nil {
		log.Fatal().Msgf("Error unmarshaling V2_FACTORY_JSON ABI: %v\n", err)
	}

	err = json.Unmarshal(V2_PAIR_JSON, &V2_PAIR)
	if err != nil {
		log.Fatal().Msgf("Error unmarshaling V2_PAIR_JSON ABI: %v\n", err)
	}

	go Loop()
}

func Loop() {
	ch := bus.Subscribe("lp_v2")
	for msg := range ch {
		if msg.RespondTo != 0 {
			continue // ignore responses
		}
		go process(msg)
	}
}

func process(msg *bus.Message) {
	switch msg.Topic {
	case "lp_v2":
		switch msg.Type {
		case "discover":
			err := discover(msg)
			msg.Respond(nil, err)
		case "get-pair":
			data, err := getPair(msg)
			msg.Respond(data, err)
		case "get-reserves":
			data, err := getReserves(msg)
			msg.Respond(data, err)
		case "get-position-status":
			data, err := getPositionStatus(msg)
			msg.Respond(data, err)
		default:
			log.Error().Msgf("lp_v2: unknown type: %v", msg.Type)
		}
	}
}
