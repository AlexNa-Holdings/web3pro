package ws

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/rs/zerolog/log"
)

func handleEthMethod(req RPCRequest, ctx *ConContext, res *RPCResponse) {
	method := strings.TrimPrefix(req.Method, "eth_")
	var err error

	o, ok := getAllowedOrigin(req.Web3ProOrigin)
	if !ok {
		res.Error = &RPCError{
			Code:    4001,
			Message: "Origin not allowed",
		}
		return
	}

	switch method {
	case "chainId":
		res.Result = fmt.Sprintf("0x%x", o.ChainId)
	case "subscribe":
		err = subscribe(req, ctx, res)
	case "unsubscribe":
		err = unsubscribe(req, ctx, res)
	case "accounts", "requestAccounts":
		res.Result = []string{}
		for _, a := range o.Addresses {
			res.Result = append(res.Result.([]string), a.String())
		}
	case "signTypedData_v4":
		err = signTypedData_v4(o, req, ctx, res)
	case "sign":
		err = sign(o, req, ctx, res)
	case "call":
		err = call(o, req, ctx, res)
	case "sendTransaction":
		err = sendTransaction(o, req, ctx, res)
	case "estimateGas":
		err = estimateGas(o, req, ctx, res)
	case "blockNumber":
		err = getBlockNumber(o, req, ctx, res)
	case "getTransactionByHash":
		err = getTransactionByHash(o, req, ctx, res)
	default:
		log.Error().Msgf("Method not found: %v", req)
	}

	if err != nil {
		log.Error().Err(err).Msgf("Error handling method: %s", req.Method)
		res.Error = &RPCError{
			Code:    4001,
			Message: "Error handling method",
		}
	}
}

func subscribe(req RPCRequest, ctx *ConContext, res *RPCResponse) error {
	params, ok := req.Params.([]any)
	if !ok {
		return fmt.Errorf("params must be an array of strings")
	}

	if len(params) < 1 {
		return fmt.Errorf("length of params must be at least 1")
	}

	stype, ok := params[0].(string)
	if !ok {
		return fmt.Errorf("params must be an array of strings")
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

	b := w.GetBlockchain(o.ChainId)
	if b == nil {
		return fmt.Errorf("blockchain not found")
	}

	sign_res := bus.Fetch("eth", "sign-typed-data-v4", &bus.B_EthSignTypedData_v4{
		Blockchain: b.Name,
		Address:    a.Address,
		TypedData:  data,
	})

	if sign_res.Error != nil {
		return fmt.Errorf("error signing typed data: %v", sign_res.Error)
	}

	res.Result = sign_res.Data.(string)
	return nil
}

func sign(o *cmn.Origin, req RPCRequest, ctx *ConContext, res *RPCResponse) error {

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

	address, ok := params[1].(string)
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

	b := w.GetBlockchain(o.ChainId)
	if b == nil {
		return fmt.Errorf("blockchain not found")
	}

	data_str, ok := params[0].(string)
	if !ok {
		return fmt.Errorf("params[1] is not a string")
	}

	sign_res := bus.Fetch("eth", "sign", &bus.B_EthSign{
		Blockchain: b.Name,
		Address:    a.Address,
		Data:       common.FromHex(data_str),
	})

	if sign_res.Error != nil {
		return fmt.Errorf("error signing typed data: %v", sign_res.Error)
	}

	res.Result = sign_res.Data.(string)
	return nil
}

func sendTransaction(o *cmn.Origin, req RPCRequest, ctx *ConContext, res *RPCResponse) error {
	w := cmn.CurrentWallet
	if w == nil {
		return fmt.Errorf("no wallet found")
	}

	params, ok := req.Params.([]any)
	if !ok {
		return fmt.Errorf("params must be an array of strings")
	}

	if len(params) < 1 {
		return fmt.Errorf("length of params must be at least 1")
	}

	tx_data, ok := params[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("params must be an array of strings")
	}

	// Get the address
	address, ok := tx_data["from"].(string)
	if !ok {
		return fmt.Errorf("from address not found")
	}

	from := w.GetAddress(address)
	if from == nil {
		return fmt.Errorf("address not found in wallet")
	}

	signer := w.GetSigner(from.Signer)
	if signer == nil {
		return fmt.Errorf("signer not found")
	}

	b := w.GetBlockchain(o.ChainId)
	if b == nil {
		return fmt.Errorf("blockchain not found")
	}

	to_s, ok := tx_data["to"].(string)
	if !ok {
		return fmt.Errorf("to address not found")
	}

	to := common.HexToAddress(to_s)

	value_s, ok := tx_data["value"].(string)
	if !ok {
		value_s = "0x00"
	}

	value := big.NewInt(0)
	value, ok = value.SetString(value_s, 0)
	if !ok {
		return fmt.Errorf("error converting value to big.Int")
	}

	data_s, ok := tx_data["data"].(string)
	if !ok {
		return fmt.Errorf("data not found")
	}
	data, err := hexutil.Decode(data_s)
	if err != nil {
		return fmt.Errorf("error decoding data: %v", err)
	}

	send_res := bus.Fetch("eth", "send-tx", &bus.B_EthSendTx{
		ChainId: b.ChainId,
		From:    from.Address,
		To:      to,
		Amount:  value,
		Data:    data,
	})

	if send_res.Error != nil {
		return fmt.Errorf("error sending transaction: %v", send_res.Error)
	}

	res.Result = send_res.Data
	return nil
}

func call(o *cmn.Origin, req RPCRequest, ctx *ConContext, res *RPCResponse) error {
	w := cmn.CurrentWallet
	if w == nil {
		return fmt.Errorf("no wallet found")
	}

	params, ok := req.Params.([]any)
	if !ok {
		return fmt.Errorf("params must be an array of strings")
	}

	if len(params) < 1 {
		return fmt.Errorf("length of params must be at least 1")
	}

	tx_data, ok := params[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("params must be an array of strings")
	}

	from := &cmn.Address{}

	// Get the address
	address, ok := tx_data["from"].(string)
	if ok {
		from = w.GetAddress(address)
	}

	b := w.GetBlockchain(o.ChainId)
	if b == nil {
		return fmt.Errorf("blockchain not found")
	}

	to_s, ok := tx_data["to"].(string)
	if !ok {
		return fmt.Errorf("to address not found")
	}

	to := common.HexToAddress(to_s)

	value_s, ok := tx_data["value"].(string)
	if !ok {
		value_s = "0x00"
	}

	value := big.NewInt(0)
	value, ok = value.SetString(value_s, 0)
	if !ok {
		return fmt.Errorf("error converting value to big.Int")
	}

	data_s, ok := tx_data["data"].(string)
	if !ok {
		return fmt.Errorf("data not found")
	}
	data, err := hexutil.Decode(data_s)
	if err != nil {
		return fmt.Errorf("error decoding data: %v", err)
	}

	send_res := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: b.ChainId,
		From:    from.Address,
		To:      to,
		Amount:  value,
		Data:    data,
	})

	if send_res.Error != nil {
		return fmt.Errorf("error sending transaction: %v", send_res.Error)
	}

	res.Result = send_res.Data
	return nil
}

func estimateGas(o *cmn.Origin, req RPCRequest, ctx *ConContext, res *RPCResponse) error {
	w := cmn.CurrentWallet
	if w == nil {
		return fmt.Errorf("no wallet found")
	}

	params, ok := req.Params.([]any)
	if !ok {
		return fmt.Errorf("params must be an array of strings")
	}

	if len(params) < 1 {
		return fmt.Errorf("length of params must be at least 1")
	}

	tx_data, ok := params[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("params must be an array of strings")
	}

	// Get the address
	address, ok := tx_data["from"].(string)
	if !ok {
		return fmt.Errorf("from address not found")
	}

	from := w.GetAddress(address)
	if from == nil {
		return fmt.Errorf("address not found in wallet")
	}

	signer := w.GetSigner(from.Signer)
	if signer == nil {
		return fmt.Errorf("signer not found")
	}

	b := w.GetBlockchain(o.ChainId)
	if b == nil {
		return fmt.Errorf("blockchain not found")
	}

	to_s, ok := tx_data["to"].(string)
	if !ok {
		return fmt.Errorf("to address not found")
	}

	to := common.HexToAddress(to_s)

	value_s, ok := tx_data["value"].(string)
	if !ok {
		value_s = "0x00"
	}

	value := big.NewInt(0)
	value, ok = value.SetString(value_s, 0)
	if !ok {
		return fmt.Errorf("error converting value to big.Int")
	}

	data_s, ok := tx_data["data"].(string)
	if !ok {
		return fmt.Errorf("data not found")
	}
	data, err := hexutil.Decode(data_s)
	if err != nil {
		return fmt.Errorf("error decoding data: %v", err)
	}

	send_res := bus.Fetch("eth", "estimate-gas", &bus.B_EthEstimateGas{
		Blockchain: b.Name,
		From:       from.Address,
		To:         to,
		Amount:     value,
		Data:       data,
	})

	if send_res.Error != nil {
		return fmt.Errorf("error sending transaction: %v", send_res.Error)
	}

	res.Result = send_res.Data
	return nil
}

func getBlockNumber(o *cmn.Origin, req RPCRequest, ctx *ConContext, res *RPCResponse) error {
	w := cmn.CurrentWallet
	if w == nil {
		return fmt.Errorf("no wallet found")
	}

	b := w.GetBlockchain(o.ChainId)
	if b == nil {
		return fmt.Errorf("blockchain not found")
	}

	send_res := bus.Fetch("eth", "block-number", &bus.B_EthBlockNumber{
		Blockchain: b.Name,
	})

	if send_res.Error != nil {
		return fmt.Errorf("error getting block number: %v", send_res.Error)
	}

	res.Result = send_res.Data
	return nil
}

func getTransactionByHash(o *cmn.Origin, req RPCRequest, ctx *ConContext, res *RPCResponse) error {
	w := cmn.CurrentWallet
	if w == nil {
		return fmt.Errorf("no wallet found")
	}

	params, ok := req.Params.([]any)
	if !ok {
		return fmt.Errorf("params must be an array of strings")
	}

	if len(params) < 1 {
		return fmt.Errorf("length of params must be at least 1")
	}

	tx_hash, ok := params[0].(string)
	if !ok {
		return fmt.Errorf("tx_hash must be a string")
	}

	b := w.GetBlockchain(o.ChainId)
	if b == nil {
		return fmt.Errorf("blockchain not found")
	}

	send_res := bus.Fetch("eth", "tx-by-hash", &bus.B_EthTxByHash{
		Blockchain: b.Name,
		Hash:       common.HexToHash(tx_hash),
	})

	if send_res.Error != nil {
		return fmt.Errorf("error getting transaction by hash: %v", send_res.Error)
	}
	res.Result = send_res.Data

	return nil
}
