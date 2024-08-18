package ws

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
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
	case "watchAsset":
		watchAssets(req)
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
	params, ok := req.Params.([]interface{})
	if !ok {
		return fmt.Errorf("invalid params: %v", req.Params)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return fmt.Errorf("wallet not found")
	}

	o := w.GetOrigin(req.Web3ProOrigin)
	if o == nil {
		return fmt.Errorf("origin not found: %v", req.Web3ProOrigin)
	}

	m, ok := params[0].(map[string]interface{})
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

func watchAssets(req RPCRequest) {
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		log.Error().Msgf("Invalid params: %v", req.Params)
		return
	}

	options, ok := params["options"].(map[string]any)
	if !ok {
		log.Error().Msgf("Invalid options: %v", params["options"])
		return
	}

	address, ok := options["address"].(string)
	if !ok {
		log.Error().Msgf("Invalid address: %v", options["address"])
		return
	}
	symbol, ok := options["symbol"].(string)
	if !ok {
		log.Error().Msgf("Invalid symbol: %v", options["symbol"])
		return
	}
	decimals, ok := options["decimals"].(float64)
	if !ok {
		log.Error().Msgf("Invalid decimals: %v", options["decimals"])
		return
	}
	//image := options["image"] // ignore

	w := cmn.CurrentWallet
	if w == nil {
		log.Error().Msg("Wallet not found")
		return
	}

	o := w.GetOrigin(req.Web3ProOrigin)
	if o == nil {
		log.Error().Msgf("Origin not found: %v", req.Web3ProOrigin)
		return
	}

	b := w.GetBlockchainById(o.ChainId)
	if b == nil {
		log.Error().Msgf("Blockchain not found: %v", o.ChainId)
		return
	}

	t := w.GetTokenByAddress(b.Name, common.HexToAddress(address))
	if t == nil {

		bus.Fetch("ui", "hail", &bus.B_Hail{
			Title: "Add Token",
			Template: `<c><w>
Do you want to add the token:
<u><b>` + symbol + `</b></u>
to your wallet?

<button text:Ok> <button text:Cancel>`,
			OnOk: func(m *bus.Message) {

				t = &cmn.Token{
					Blockchain: b.Name,
					Address:    common.HexToAddress(address),
					Symbol:     symbol,
					Decimals:   int(decimals),
				}
				w.Tokens = append(w.Tokens, t)
				err := w.Save()
				if err != nil {
					log.Error().Err(err).Msg("Failed to save wallet")
					return
				}
				bus.Send("ui", "notify", "Token added to wallet")
			},
		})

	} else {
		bus.Send("ui", "notify", "Token already in wallet")
	}
}