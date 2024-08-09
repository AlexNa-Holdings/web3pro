package ws

import (
	"strings"
)

func handleEthMethod(req RPCRequest, res map[string]interface{}) {

	method := strings.TrimPrefix(req.Method, "eth_")

	switch method {
	case "chainId":
		res["result"] = "0x1"
	}

}
