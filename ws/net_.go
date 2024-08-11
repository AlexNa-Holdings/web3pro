package ws

import (
	"strings"
)

func handleNetMethod(req RPCRequest, ctx *ConContext, res map[string]interface{}) {

	method := strings.TrimPrefix(req.Method, "net_")

	switch method {
	case "version":
		res["result"] = "0x1"
	}
}
