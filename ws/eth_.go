package ws

import (
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

func handleEthMethod(req RPCRequest, ctx *ConContext, res *RPCResponse) {
	method := strings.TrimPrefix(req.Method, "eth_")

	switch method {
	case "chainId":
		requestChainId(req, ctx, res)
	case "subscribe":
		subscribe(req, ctx, res)
	case "requestAccounts":
		requestAccounts(req, ctx, res)
	default:
		log.Error().Msgf("Method not found: %v", req)
		res.Error = &RPCError{
			Code:    4001,
			Message: "Method not found",
		}
	}
}

func requestChainId(req RPCRequest, _ *ConContext, res *RPCResponse) {
	id := 1

	w := cmn.CurrentWallet
	if w != nil {
		origin := w.GetOrigin(req.Web3ProOrigin)
		if origin != nil {
			id = origin.ChainId
		} else {
			b := w.GetBlockchain(w.CurrentChain)
			if b != nil {
				id = b.ChainId
			}
		}
	}
	res.Result = fmt.Sprintf("0x%x", id)
}

func requestAccounts(req RPCRequest, _ *ConContext, res *RPCResponse) {
	w := cmn.CurrentWallet

	origin := w.GetOrigin(req.Web3ProOrigin)
	if origin == nil {
		bus.Fetch("ui", "hail", &bus.B_Hail{
			Title: "Connect Web Application",
			Template: `<c><w>
Allow to connect to this web application:

<u><b>` + req.Web3ProOrigin + `</b></u>

and use the current address?

<button text:Ok> <button text:Cancel>`,
			OnOk: func(m *bus.Message) {

				chain_id := 1
				b := w.GetBlockchain(w.CurrentChain)
				if b != nil {
					chain_id = b.ChainId
				}

				origin = &cmn.Origin{
					URL:       req.Web3ProOrigin,
					ChainId:   chain_id,
					Addresses: []common.Address{w.CurrentAddress},
				}

				w.AddOrigin(origin)
				err := w.Save()
				if err != nil {
					log.Error().Err(err).Msg("Failed to save wallet")
					bus.Send("ui", "notify", "Failed to save wallet")
				}
				bus.Send("ui", "remove-hail", m)
			}})
	}

	if origin == nil || len(origin.Addresses) == 0 {
		res.Result = []string{}
		res.Error = &RPCError{
			Code:    4001,
			Message: "User rejected request",
		}
		return
	}

	res.Result = []string{}
	for _, a := range origin.Addresses {
		res.Result = append(res.Result.([]string), a.String())
	}
}

func subscribe(req RPCRequest, ctx *ConContext, res *RPCResponse) {

	if len(req.Params) < 1 {
		res.Error = &RPCError{
			Code:    4001,
			Message: "Invalid params",
		}
		return

	}

	stype := req.Params[0].(string)

	switch stype {
	case "chainChanged", "accountsChanged":
		id := ctx.SM.addSubscription(req.Web3ProOrigin, stype, nil)
		res.Result = fmt.Sprintf("0x%x", id)
	default:
		log.Error().Msgf("Invalid subscription type: %v", req)
		res.Error = &RPCError{
			Code:    4001,
			Message: "Invalid params",
		}
	}
}
