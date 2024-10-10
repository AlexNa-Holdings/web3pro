package lp_v3

import (
	_ "embed"
	"encoding/json"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/ava-labs/coreth/accounts/abi"
	"github.com/rs/zerolog/log"
)

//go:embed NonfungiblePositionManager.json
var V3_ABI_JSON []byte
var V3_ABI abi.ABI

func Init() {
	err := json.Unmarshal(V3_ABI_JSON, &V3_ABI)
	if err != nil {
		log.Fatal().Msgf("Error unmarshaling V3_ABI_JSON ABI: %v\n", err)
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
		}
	}
}
