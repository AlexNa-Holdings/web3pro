package signer

import (
	"github.com/AlexNa-Holdings/web3pro/bus"
)

func Init() {
	go Loop()
}

func Loop() {
	ch := bus.Subscribe("signer")
	for {
		select {
		case msg := <-ch:
			if msg.RespondTo != 0 {
				continue // ignore responses
			}

			switch msg.Type {
			case "add":
			}
		}
	}
}
