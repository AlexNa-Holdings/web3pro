package ws

import (
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/rs/zerolog/log"
)

func handleEthMethod(req RPCRequest, ctx *ConContext, res *RPCResponse) {
	method := strings.TrimPrefix(req.Method, "eth_")

	switch method {
	case "chainId":
		requestChainId(req, ctx, res)
	case "subscribe":
		subscribe(req, ctx, res)
	case "accounts", "requestAccounts":
		accounts(req, ctx, res)
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

func accounts(req RPCRequest, _ *ConContext, res *RPCResponse) {
	if o, ok := getAllowedOrigin(req.Web3ProOrigin); ok {
		res.Result = []string{}
		for _, a := range o.Addresses {
			res.Result = append(res.Result.([]string), a.String())

		}
	} else {
		res.Result = []string{}
		res.Error = &RPCError{
			Code:    4001,
			Message: "User rejected request",
		}
	}
}

func subscribe(req RPCRequest, ctx *ConContext, res *RPCResponse) {

	params, ok := req.Params.([]string)
	if !ok {
		res.Error = &RPCError{
			Code:    4001,
			Message: "Params must be an array of strings",
		}
		return
	}

	if len(params) < 1 {
		res.Error = &RPCError{
			Code:    4001,
			Message: "Invalid params",
		}
		return
	}

	stype := params[0]

	switch stype {
	case "chainChanged", "accountsChanged":
		res.Result = ctx.SM.addSubscription(req.Web3ProOrigin, stype, nil)
	default:
		log.Error().Msgf("Invalid subscription type: %v", req)
		res.Error = &RPCError{
			Code:    4001,
			Message: "Invalid params",
		}
	}
}
