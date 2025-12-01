package staking

import (
	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/rs/zerolog/log"
)

func Init() {
	go Loop()
}

func Loop() {
	ch := bus.Subscribe("staking")
	for msg := range ch {
		if msg.RespondTo != 0 {
			continue // ignore responses
		}
		go process(msg)
	}
}

func process(msg *bus.Message) {
	switch msg.Topic {
	case "staking":
		switch msg.Type {
		case "get-balance":
			data, err := getBalance(msg)
			msg.Respond(data, err)
		case "get-pending":
			data, err := getPending(msg)
			msg.Respond(data, err)
		case "get-delegations":
			data, err := getDelegations(msg)
			msg.Respond(data, err)
		default:
			log.Error().Msgf("staking: unknown type: %v", msg.Type)
		}
	}
}
