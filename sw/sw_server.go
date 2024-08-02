package sw

import "github.com/AlexNa-Holdings/web3pro/sw/mnemonics"

func Init() {
	go mnemonics.Loop()
}
