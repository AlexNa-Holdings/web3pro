package hw

import (
	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/hw/trezor"
)

func Init() {
	go trezor.Loop()
	go Loop()
}

func Loop() {
	ch := bus.Subscribe("hw", "usb")
	for msg := range ch {
		if msg.RespondTo != 0 {
			continue // ignore responses
		}

		switch msg.Type {
		case "init":
		}
	}
}
