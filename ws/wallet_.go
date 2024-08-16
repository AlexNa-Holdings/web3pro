package ws

import (
	"strings"

	"github.com/rs/zerolog/log"
)

func handleWalletMethod(req RPCRequest, ctx *ConContext, res *RPCResponse) {

	method := strings.TrimPrefix(req.Method, "wallet_")

	switch method {
	case "switchEthereumChain":
		//TODO
	case "requestPermissions":
		// TODO
	default:
		log.Error().Msgf("Method not found: %v", req)
		res.Error = &RPCError{
			Code:    4001,
			Message: "Method not found",
		}
	}
}
