package hw

import (
	"github.com/AlexNa-Holdings/web3pro/hw/ledger"
	"github.com/AlexNa-Holdings/web3pro/hw/trezor"
)

func Init() {
	go trezor.Loop()
	go ledger.Loop()
}
