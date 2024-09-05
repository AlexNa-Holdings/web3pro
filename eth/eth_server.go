package eth

import (
	"fmt"
	"sync"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
)

type con struct {
	*ethclient.Client
	URL string
}

var cons map[int]*con // chainId -> client
var consMutex = sync.Mutex{}

func Init() {
	LoadABIs()
	go Loop()
}

func Loop() {
	ch := bus.Subscribe("eth", "wallet")
	for msg := range ch {
		if msg.RespondTo != 0 {
			continue // ignore responses
		}
		go process(msg)
	}
}

func process(msg *bus.Message) {
	switch msg.Topic {
	case "eth":
		switch msg.Type {
		case "send":
			err := send(msg)
			msg.Respond(nil, err)
		case "send-tx":
			hash, err := sendTx(msg)
			msg.Respond(hash, err)
		case "sign-typed-data-v4":
			sig, err := signTypedDataV4(msg)
			msg.Respond(sig, err)
		}
	case "wallet":
		switch msg.Type {
		case "open":
			initConnections()
		case "saved":
			updateConnections()
		}
	}
}

func initConnections() {
	consMutex.Lock()
	defer consMutex.Unlock()

	for _, c := range cons {
		c.Close()
	}
	cons = make(map[int]*con)

	w := cmn.CurrentWallet
	if w == nil {
		return
	}

	for _, b := range w.Blockchains {
		if _, ok := cons[b.ChainID]; !ok {
			openClient_locked(b)
		}
	}
}

func updateConnections() {
	consMutex.Lock()
	defer consMutex.Unlock()

	w := cmn.CurrentWallet
	if w == nil {
		return
	}

	vetted := make(map[int]bool)
	for _, b := range w.Blockchains {
		vetted[b.ChainID] = true

		c, ok := cons[b.ChainID]
		if !ok {
			openClient_locked(b)
		} else {

			if c.URL != b.Url {
				//reconnect
				cons[b.ChainID].Close()
				bus.Send("eth", "disconnected", b.ChainID)
				openClient_locked(b)
			}
		}
	}

	to_delete := []int{}
	for c := range cons {
		if _, ok := vetted[c]; !ok {
			to_delete = append(to_delete, c)
		}
	}

	for _, c := range to_delete {
		cons[c].Close()
		delete(cons, c)
	}

}

func openClient_locked(b *cmn.Blockchain) error {
	client, err := ethclient.Dial(b.Url)
	if err != nil {
		log.Error().Msgf("OpenClient: Cannot dial to (%s). Error:(%v)", b.Url, err)
		return err
	}
	cons[b.ChainID] = &con{client, b.Url}
	log.Trace().Msgf("OpenClient: Client opened to (%s)", b.Url)
	bus.Send("eth", "connected", b.ChainID)
	return nil
}

func getEthClient(b *cmn.Blockchain) (*ethclient.Client, error) {
	consMutex.Lock()
	defer consMutex.Unlock()

	c, ok := cons[b.ChainID]
	if !ok {
		return nil, fmt.Errorf("OpenClient: Client not found for chainId (%d)", b.ChainID)
	}

	if c.Client == nil {
		return nil, fmt.Errorf("OpenClient: Client is nil for chainId (%d)", b.ChainID)
	}

	return c.Client, nil
}
