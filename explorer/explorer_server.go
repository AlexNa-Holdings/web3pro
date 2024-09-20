package explorer

import (
	"errors"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

func Init() {
	go Loop()
}

type Explorer interface {
	DownloadContract(w *cmn.Wallet, b *cmn.Blockchain, contract common.Address) error
}

func Loop() {
	ch := bus.Subscribe("explorer")
	for msg := range ch {
		if msg.RespondTo != 0 {
			continue // ignore responses
		}
		go process(msg)
	}
}

func process(msg *bus.Message) {
	w := cmn.CurrentWallet
	if w == nil {
		msg.Respond(nil, errors.New("no wallet"))
		return
	}

	switch msg.Topic {
	case "download-contract":
		m, ok := msg.Data.(*bus.B_ExplorerDownloadContract)
		if !ok {
			log.Error().Msg("Loop: Invalid explorer download-contract data")
			return
		}

		err := download(m)
		msg.Respond(nil, err)
	}
}

func download(m *bus.B_ExplorerDownloadContract) error {
	w := cmn.CurrentWallet
	if w == nil {
		return errors.New("no wallet")
	}

	b := w.GetBlockchain(m.Blockchain)
	if b == nil {
		return errors.New("no blockchain")
	}

	var ex Explorer

	switch b.ExplorerApiType {
	case "etherscan":
		ex = &EtherScanAPI{}
	case "blockscout":
		ex = &BlockscoutAPI{}
	}

	if ex == nil {
		return errors.New("no explorer")
	}

	err := ex.DownloadContract(w, b, m.Address)
	if err != nil {
		log.Error().Err(err).Msg("Error downloading contract")
		return err
	}

	err = w.Save()
	if err != nil {
		log.Error().Err(err).Msg("Error saving wallet")
		return err
	}

	return nil
}
