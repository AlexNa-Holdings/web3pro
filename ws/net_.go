package ws

import (
	"strings"
)

func handleNetMethod(req RPCRequest, ctx *ConContext, res *RPCResponse) {

	method := strings.TrimPrefix(req.Method, "net_")

	switch method {
	case "version":
		res.Result = "0x1"
	}
}
