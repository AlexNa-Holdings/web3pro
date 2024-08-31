package ws

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/rs/zerolog/log"
)

func handleEthMethod(req RPCRequest, ctx *ConContext, res *RPCResponse) {
	method := strings.TrimPrefix(req.Method, "eth_")
	var err error

	switch method {
	case "chainId":
		if o, ok := getAllowedOrigin(req.Web3ProOrigin); ok {
			res.Result = fmt.Sprintf("0x%x", o.ChainId)
		} else {
			res.Result = "0x1"
		}
	case "subscribe":
		if _, ok := getAllowedOrigin(req.Web3ProOrigin); ok {
			err = subscribe(req, ctx, res)
		} else {
			err = fmt.Errorf("origin not allowed")
		}
	case "unsubscribe":
		err = unsubscribe(req, ctx, res)
	case "accounts", "requestAccounts":
		if o, ok := getAllowedOrigin(req.Web3ProOrigin); ok {
			res.Result = []string{}
			for _, a := range o.Addresses {
				res.Result = append(res.Result.([]string), a.String())

			}
		} else {
			err = fmt.Errorf("origin not allowed")
		}
	case "signTypedData_v4":
		if o, ok := getAllowedOrigin(req.Web3ProOrigin); ok {
			err = signTypedData_v4(o, req, ctx, res)
		} else {
			err = fmt.Errorf("origin not allowed")
		}

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

func signTypedData_v4(o *cmn.Origin, req RPCRequest, ctx *ConContext, res *RPCResponse) error {

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

	a := w.GetAddress(address)
	if a == nil {
		return fmt.Errorf("address not found in wallet")
	}

	signer := w.GetSigner(a.Signer)
	if signer == nil {
		return fmt.Errorf("signer not found")
	}

	var data apitypes.TypedData

	if p1, ok := params[1].(string); ok {
		err := json.Unmarshal([]byte(p1), &data)
		if err != nil {
			return fmt.Errorf("error unmarshalling typed data: %v", err)
		}
	} else if p1Map, ok := params[1].(map[string]interface{}); ok {
		// fix version type
		if dmn, ok := p1Map["domain"].(map[string]interface{}); ok {
			ver, ok := dmn["version"].(float64)
			if ok {
				dmn["version"] = fmt.Sprintf("%v", ver)
			}
		}

		// Marshal the map into JSON
		mapJSON, err := json.Marshal(p1Map)
		if err != nil {
			return fmt.Errorf("error marshalling map to JSON: %v", err)
		}

		log.Debug().Msgf("mapJSON: %v", string(mapJSON))

		// Unmarshal the JSON into the EIP712_TypedData struct
		err = json.Unmarshal(mapJSON, &data)
		if err != nil {
			return fmt.Errorf("error unmarshalling map to typed data: %v", err)
		}
	} else {
		return fmt.Errorf("params[1] is neither a string nor a map[string]interface{}")
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
