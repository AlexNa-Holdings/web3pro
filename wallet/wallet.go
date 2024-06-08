package wallet

import "github.com/AlexNa-Holdings/web3pro/cmn"

type Wallet struct {
}

var CurrentWallet *Wallet

func OpenWallet(name string) error {
	var err error

	CurrentWallet, err = OpenFromFile(cmn.DataFolder + "/" + name + ".wallet")

	return err
}

func OpenFromFile(file string) (*Wallet, error) {
	var err error

	w := &Wallet{}

	return w, err
}
