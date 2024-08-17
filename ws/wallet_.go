package ws

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/rs/zerolog/log"
)

func handleWalletMethod(req RPCRequest, ctx *ConContext, res *RPCResponse) {
	var err error
	method := strings.TrimPrefix(req.Method, "wallet_")

	switch method {
	case "switchEthereumChain":
		err = switchEthereumChain(req)
	case "requestPermissions":
		// TODO
	default:
		log.Error().Msgf("Method not found: %v", req)
		res.Error = &RPCError{
			Code:    4001,
			Message: "Method not found",
		}
	}

	if err != nil {
		log.Error().Msgf("Error: %v", err)
		res.Error = &RPCError{
			Code:    4000,
			Message: err.Error(),
		}
	}
}

func switchEthereumChain(req RPCRequest) error {

	w := cmn.CurrentWallet
	if w == nil {
		return fmt.Errorf("wallet not found")
	}

	o := w.GetOrigin(req.Web3ProOrigin)
	if o == nil {
		return fmt.Errorf("origin not found: %v", req.Web3ProOrigin)
	}

	m, ok := req.Params[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid params: %v", req.Params)
	}

	mID, ok := m["chainId"]
	if !ok {
		return fmt.Errorf("invalid chainId: %v", req.Params)
	}

	sID, ok := mID.(string)
	if !ok {
		return fmt.Errorf("invalid chainId: %v", mID)
	}

	chainID, err := strconv.ParseInt(sID, 0, 64)
	if err != nil {
		return fmt.Errorf("invalid chainId: %v", sID)
	}

	b := w.GetBlockchainById(int(chainID))
	if b == nil {
		return fmt.Errorf("blockchain not found: %v", chainID)
	}

	schain := fmt.Sprintf("%s (%d)", b.Name, b.ChainId)

	bus.Fetch("ui", "hail", &bus.B_Hail{
		Title: "Switch Chain",
		Template: `<c><w>
Do you want to swith the blockchain for webb application:
<u><b>` + cmn.GetHostName(req.Web3ProOrigin) + `</b></u>
to :
<u><b>` + schain + `</b></u>

<button text:Ok> <button text:Cancel>`,
		OnOk: func(m *bus.Message) {
			o.ChainId = int(chainID)
			err := w.Save()
			if err != nil {
				log.Error().Err(err).Msg("Failed to save wallet")
				bus.Send("ui", "notify", "Failed to save wallet")
			}
			bus.Send("ui", "remove-hail", m)
		}})

	return nil
}
