package ws

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const WEB3PRO_PORT = 9323 // WEB3 mnemonics

type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      json.RawMessage `json:"id"`
}

func Init() {
	http.HandleFunc("/ws", web3Handler)

	go func() {

		log.Trace().Msgf("ws server started on port %d", WEB3PRO_PORT)
		err := http.ListenAndServe(":"+strconv.Itoa(WEB3PRO_PORT), nil)
		if err != nil {
			log.Fatal().Err(err).Msgf("WS server failed to start on port %d", WEB3PRO_PORT)
		}
	}()

}

// Upgrade HTTP connection to a WebSocket connection
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin
		return true
	},
}

func web3Handler(w http.ResponseWriter, r *http.Request) {

	if r.URL.Query().Get("identity") != "web3pro-extension" {
		log.Error().Msgf("Atteempt connecting to WS with invalid identity: %s", r.URL.Query().Get("identity"))
		http.Error(w, "Invalid token", http.StatusForbidden)
	}

	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error()
		return
	}
	defer conn.Close()

	for {
		// Read message from browser
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Error().Msgf("Read error: %v", err)
			break
		}

		if msgType != websocket.TextMessage {
			log.Trace().Msgf("Received non-text message: %d", msgType)
			continue
		}

		// Print the message to the console
		log.Trace().Msgf("Received: %s", msg)

		// Parse the JSON-RPC message
		var rpcReq RPCRequest
		err = json.Unmarshal(msg, &rpcReq)
		if err != nil {
			log.Printf("JSON parse error: %v", err)
			continue
		}

		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      rpcReq.ID,
		}

		// Dispatch based on method prefix
		switch {
		case strings.HasPrefix(rpcReq.Method, "eth_"):
			handleEthMethod(rpcReq, response)
		default:
			log.Printf("Unknown method: %s", rpcReq.Method)
			// Handle unknown methods or send an error response
			response["error"] = map[string]interface{}{
				"code":    -32601,
				"message": "Method not found",
			}
		}
		sendResponse(conn, response)
	}
}

func sendResponse(conn *websocket.Conn, response map[string]interface{}) {
	respBytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return
	}
	err = conn.WriteMessage(websocket.TextMessage, respBytes)
	if err != nil {
		log.Printf("Write error: %v", err)
	}
}
