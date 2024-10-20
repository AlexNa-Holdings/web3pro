package eth

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
)

type con struct {
	*ethclient.Client
	URL string
}

var cons map[int]*con = make(map[int]*con) // chainId -> client
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
		case "call":
			data, err := call(msg)
			msg.Respond(data, err)
		case "multi-call":
			data, err := multiCall(msg)
			msg.Respond(data, err)
		case "sign-typed-data-v4":
			sig, err := signTypedDataV4(msg)
			msg.Respond(sig, err)
		case "sign":
			sig, err := sign(msg)
			msg.Respond(sig, err)
		case "estimate-gas":
			gas, err := estimateGas(msg)
			msg.Respond(gas, err)
		case "block-number":
			blockNumber, err := blockNumber(msg)
			msg.Respond(blockNumber, err)
		case "get-tx-by-hash":
			tx, err := getTxByHash(msg)
			msg.Respond(tx, err)
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
		if _, ok := cons[b.ChainId]; !ok {
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
		vetted[b.ChainId] = true

		c, ok := cons[b.ChainId]
		if !ok {
			openClient_locked(b)
		} else {

			if c.URL != b.Url {
				//reconnect
				cons[b.ChainId].Close()
				bus.Send("eth", "disconnected", b.ChainId)
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
	cons[b.ChainId] = &con{client, b.Url}
	log.Trace().Msgf("OpenClient: Client opened to (%s)", b.Url)
	bus.Send("eth", "connected", b.ChainId)
	return nil
}

func getEthClient(b *cmn.Blockchain) (*ethclient.Client, error) {
	consMutex.Lock()
	defer consMutex.Unlock()

	c, ok := cons[b.ChainId]
	if !ok {
		return nil, fmt.Errorf("OpenClient: Client not found for chainId (%d)", b.ChainId)
	}

	if c.Client == nil {
		return nil, fmt.Errorf("OpenClient: Client is nil for chainId (%d)", b.ChainId)
	}

	return c.Client, nil
}

func blockNumber(msg *bus.Message) (string, error) {
	req, ok := msg.Data.(*bus.B_EthBlockNumber)
	if !ok {
		return "", fmt.Errorf("invalid tx: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return "", fmt.Errorf("no wallet")
	}

	b := w.GetBlockchainByName(req.Blockchain)
	if b == nil {
		return "", fmt.Errorf("blockchain not found: %v", req.Blockchain)
	}

	c, err := getEthClient(b)
	if err != nil {
		return "", err
	}

	blockNumber, err := c.BlockNumber(context.Background())
	if err != nil {
		return "", err
	}

	n_hex := fmt.Sprintf("0x%x", blockNumber)

	return n_hex, nil
}

func getTxByHash(msg *bus.Message) (*bus.B_EthTxByHash_Response, error) {
	req, ok := msg.Data.(*bus.B_EthTxByHash)
	if !ok {
		return nil, fmt.Errorf("invalid tx: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("no wallet")
	}

	b := w.GetBlockchainByName(req.Blockchain)
	if b == nil {
		return nil, fmt.Errorf("blockchain not found: %v", req.Blockchain)
	}

	c, err := getEthClient(b)
	if err != nil {
		return nil, err
	}

	tx, pending, err := c.TransactionByHash(context.Background(), req.Hash)
	if err != nil {
		return nil, err
	}

	// Get block details if the transaction is confirmed
	var blockHash common.Hash
	var blockNumber *big.Int
	if !pending {
		txReceipt, err := c.TransactionReceipt(context.Background(), req.Hash)
		if err != nil {
			return nil, err
		}
		blockHash = txReceipt.BlockHash
		blockNumber = txReceipt.BlockNumber
	} else {
		blockHash = common.Hash{} // Placeholder for pending transactions
		blockNumber = big.NewInt(0)
	}

	from, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender: %v", err)
	}

	to := ""
	if tx.To() != nil {
		to = tx.To().Hex()
	}

	r, s, v := tx.RawSignatureValues()

	resp := bus.B_EthTxByHash_Response{
		BlockHash:        blockHash.Hex(),
		BlockNumber:      blockNumber.String(),
		ChainID:          b.ChainId,
		From:             from.Hex(),
		Gas:              fmt.Sprintf("%d", tx.Gas()),
		GasPrice:         tx.GasPrice().String(),
		Hash:             tx.Hash().Hex(),
		Input:            common.Bytes2Hex(tx.Data()),
		Nonce:            fmt.Sprintf("%d", tx.Nonce()),
		To:               to,
		TransactionIndex: "", // Needs to be filled if available
		Value:            tx.Value().String(),
		V:                v.String(),
		R:                r.String(),
		S:                s.String(),
	}

	if !pending {
		// Fill in the transaction index if the transaction is mined
		txReceipt, err := c.TransactionReceipt(context.Background(), req.Hash)
		if err != nil {
			return nil, err
		}
		resp.TransactionIndex = fmt.Sprintf("%d", txReceipt.TransactionIndex)
	}

	return &resp, nil
}
