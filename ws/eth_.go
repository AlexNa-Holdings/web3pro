package ws

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
)

func handleEthMethod(req RPCRequest, ctx *ConContext, res *RPCResponse) {
	w := cmn.CurrentWallet

	method := strings.TrimPrefix(req.Method, "eth_")

	switch method {
	case "chainId":
		res.Result = "0x1"
	case "requestAccounts":

		origin := w.GetOrigin(req.Web3ProOrigin)
		if origin == nil {
			bus.Fetch("ui", "hail", &bus.B_Hail{
				Title: "Connect Web Application",
				Template: `<c><w>
Allow to connect to this web application:

<u><b>` + req.Web3ProOrigin + `</b></u>

and use the current address?

<button text:Ok> <button text:Cancel>`,
				OnOk: func(h *bus.B_Hail) {
					origin = &cmn.Origin{
						URL:       req.Web3ProOrigin,
						Addresses: []common.Address{w.CurrentAddress},
					}

					w.AddOrigin(origin)
					bus.Send("ui", "remove-hail", h)
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

}
