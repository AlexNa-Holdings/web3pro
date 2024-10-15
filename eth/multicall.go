package eth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

const THRESHOLD time.Duration = 2 * time.Second

type MultiCall struct {
	Queue             []*bus.Message
	Mutex             sync.Mutex
	LastSendTime      time.Time
	Timer             *time.Timer
	ChainId           int
	MulticallContract common.Address
}

func NewMultiCall(chain_id int, contract common.Address) *MultiCall {
	return &MultiCall{
		Queue:             make([]*bus.Message, 0),
		Mutex:             sync.Mutex{},
		LastSendTime:      time.Now(),
		Timer:             nil,
		ChainId:           chain_id,
		MulticallContract: contract,
	}
}

func (mc *MultiCall) Add(msg *bus.Message) {
	mc.Mutex.Lock()
	defer mc.Mutex.Unlock()

	mc.Queue = append(mc.Queue, msg)
	if time.Since(mc.LastSendTime) > THRESHOLD {
		mc._send()
	} else {
		if mc.Timer == nil {
			mc.Timer = time.AfterFunc(THRESHOLD, mc._send)
		}
	}
}

func (mc *MultiCall) _send() {
	mc.LastSendTime = time.Now()

	if mc.Timer != nil {
		mc.Timer.Stop()
		mc.Timer = nil
	}

	if len(mc.Queue) == 0 {
		return
	}

	w := cmn.CurrentWallet
	if w == nil {
		log.Error().Msg("multicall: no wallet")
		return
	}

	if len(mc.Queue) == 1 {

		msg := mc.Queue[0]
		mc.Queue = mc.Queue[0:0]

		req, ok := msg.Data.(*bus.B_EthCall)
		if !ok {
			log.Error().Msgf("multicall: invalid tx: %v", msg.Data)
			msg.Respond("", fmt.Errorf("invalid tx: %v", msg.Data))
			return
		}

		from := w.GetAddress(req.From.String())
		if from == nil {
			log.Error().Msgf("multicall: address from not found: %v", req.From)
			msg.Respond("", fmt.Errorf("address from not found: %v", req.From))
			return
		}

		c, ok := cons[mc.ChainId]
		if !ok {
			log.Error().Msgf("multicall: Client not found for chainId: %d", mc.ChainId)
			msg.Respond("", fmt.Errorf("client not found for chainId: %d", mc.ChainId))
			return
		}

		call_msg := ethereum.CallMsg{
			To:    &req.To,
			From:  from.Address,
			Value: req.Amount,
			Data:  req.Data,
		}

		output, err := c.CallContract(context.Background(), call_msg, nil)
		if err != nil {
			log.Error().Msgf("call: Cannot call contract. Error:(%v)", err)
			msg.Respond("", err)
			return
		}

		hex_data := fmt.Sprintf("0x%x", output)
		msg.Respond(hex_data, nil)
		return
	}

	// multi call
	callArgs := []interface{}{}

	for _, msg := range mc.Queue {
		req, ok := msg.Data.(*bus.B_EthCall)
		if !ok {
			log.Error().Msgf("multicall: invalid tx: %v", msg.Data)
			msg.Respond("", fmt.Errorf("invalid tx: %v", msg.Data))
			return // should not happen
		}

		callArgs = append(callArgs, struct {
			Target   common.Address
			CallData []byte
		}{
			Target:   req.To,
			CallData: req.Data,
		})
	}

	data, err := MULTICALL2_ABI.Pack("aggregate", callArgs)
	if err != nil {
		log.Error().Msgf("multicall: Cannot pack multicall data. Error:(%v)", err)
		mc.Queue = mc.Queue[0:0]
		return
	}

	call_msg := ethereum.CallMsg{
		To:   &mc.MulticallContract,
		Data: data,
	}

	// Unpack the return data
	var returnData [][]byte
	err = MULTICALL2_ABI.UnpackIntoInterface(&returnData, "aggregate", call_msg.Data)
	if err != nil {
		log.Error().Msgf("multicall: Cannot unpack multicall data. Error:(%v)", err)
		mc.Queue = mc.Queue[0:0]
		return
	}

	for i, data := range returnData {
		msg := mc.Queue[i]
		hex_data := fmt.Sprintf("0x%x", data)
		msg.Respond(hex_data, nil)
	}

	log.Trace().Msgf("multicall: multicall success %d calls", len(mc.Queue))

	mc.Queue = mc.Queue[0:0]

}
