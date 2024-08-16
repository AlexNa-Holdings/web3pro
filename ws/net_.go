package ws

import (
	"strings"

	"github.com/rs/zerolog/log"
)

func handleNetMethod(req RPCRequest, ctx *ConContext, res *RPCResponse) {

	method := strings.TrimPrefix(req.Method, "net_")

	switch method {
	case "version":
		res.Result = "0x1"
	default:
		log.Error().Msgf("Method not found: %v", req)
		res.Error = &RPCError{
			Code:    4001,
			Message: "Method not found",
		}
	}
}
