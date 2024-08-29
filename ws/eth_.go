package ws

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/EIP"
	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

func handleEthMethod(req RPCRequest, ctx *ConContext, res *RPCResponse) {
	method := strings.TrimPrefix(req.Method, "eth_")
	var err error

	switch method {
	case "chainId":
		err = requestChainId(req, ctx, res)
	case "subscribe":
		err = subscribe(req, ctx, res)
	case "unsubscribe":
		err = unsubscribe(req, ctx, res)
	case "accounts", "requestAccounts":
		err = accounts(req, ctx, res)
	case "signTypedData_v4":
		err = signTypedData_v4(req, ctx, res)

	default:
		log.Error().Msgf("Method not found: %v", req)
	}

	if err != nil {
		log.Error().Err(err).Msgf("Error handling method: %v", req)
		res.Error = &RPCError{
			Code:    4001,
			Message: "Error handling method",
		}
	}
}

func requestChainId(req RPCRequest, _ *ConContext, res *RPCResponse) error {
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
	return nil
}

func accounts(req RPCRequest, _ *ConContext, res *RPCResponse) error {
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

	return nil
}

func subscribe(req RPCRequest, ctx *ConContext, res *RPCResponse) error {

	params, ok := req.Params.([]any)
	if !ok {
		return fmt.Errorf("Params must be an array of strings")
	}

	if len(params) < 1 {
		return fmt.Errorf("Length of params must be at least 1")
	}

	stype, ok := params[0].(string)
	if !ok {
		return fmt.Errorf("Params must be an array of strings")
	}

	switch stype {
	case "chainChanged", "accountsChanged":
		res.Result = ctx.SM.addSubscription(req.Web3ProOrigin, stype, nil)
	default:
		return fmt.Errorf("Invalid subscription type")
	}

	return nil
}

func unsubscribe(req RPCRequest, ctx *ConContext, res *RPCResponse) error {

	params, ok := req.Params.([]any)
	if !ok {
		return fmt.Errorf("Params must be an array of strings")
	}

	if len(params) < 1 {
		return fmt.Errorf("Length of params must be at least 1")
	}

	for _, p := range params {
		id, ok := p.(string)
		if !ok {
			return fmt.Errorf("Params must be an array of strings")
		}

		ctx.SM.removeSubscription(req.Web3ProOrigin, id)
	}

	return nil
}

func signTypedData_v4(req RPCRequest, ctx *ConContext, res *RPCResponse) error {

	w := cmn.CurrentWallet
	if w == nil {
		return fmt.Errorf("no wallet found")
	}

	params, ok := req.Params.([]any)
	if !ok {
		return fmt.Errorf("params must be an array of strings")
	}

	if len(params) < 2 {
		return fmt.Errorf("length of params must be at least 2")
	}

	address, ok := params[0].(string)
	if !ok {
		return fmt.Errorf("first param must be an address")
	}

	o := w.GetOrigin(req.Web3ProOrigin)
	if o == nil {
		return fmt.Errorf("no origin found")
	}

	if !o.IsAllowed(common.HexToAddress(address)) {
		return fmt.Errorf("address not found in origin")
	}

	a := w.GetAddress(address)
	if a == nil {
		return fmt.Errorf("address not found in wallet")
	}

	signer := w.GetSigner(a.Signer)
	if signer == nil {
		return fmt.Errorf("signer not found")
	}

	var data EIP.EIP712_TypedData
	err := json.Unmarshal([]byte(params[1].(string)), &data)
	if err != nil {
		return fmt.Errorf("error unmarshalling typed data: %v", err)
	}

	sign_res := bus.Fetch("signer", "sign-typed-data-v4", &bus.B_SignerSignTypedData_v4{
		Type:      signer.Type,
		Name:      signer.Name,
		MasterKey: signer.MasterKey,
		Address:   a.Address,
		Path:      a.Path,
		TypedData: data,
	})

	if sign_res.Error != nil {
		return fmt.Errorf("error signing typed data: %v", sign_res.Error)
	}

	res.Result = sign_res.Data.(string)
	return nil
}
