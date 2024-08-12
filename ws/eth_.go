package ws

import (
	"strings"
)

func handleEthMethod(req RPCRequest, ctx *ConContext, res *RPCResponse) {

	method := strings.TrimPrefix(req.Method, "eth_")

	switch method {
	case "chainId":
		res.Result = "0x1"
	case "requestAccounts":
		res.Result = []string{"0xe328b70d1DB5c556234cade0dF86b3afBF56DD32"}
	}

}
