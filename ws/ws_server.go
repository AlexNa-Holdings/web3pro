package ws

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const WEB3PRO_PORT = 9323 // WEB3 mnemonics

type ConContext struct {
	Agent string
}

var connections = make([]*ConContext, 0)
var connectionsMutex = sync.Mutex{}

type RPCRequest struct {
	JSONRPC       string        `json:"jsonrpc"`
	ID            int64         `json:"id"`
	Method        string        `json:"method"`
	Params        []interface{} `json:"params"` // Params as a slice of interface{}
	Web3ProOrigin string        `json:"__web3proOrigin,omitempty"`
}

type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Result  interface{} `json:"result"`
	Error   interface{} `json:"error"`
}

func Init() {
	go loop()
}

func loop() {
	ch := bus.Subscribe("wallet", "ws")
	for msg := range ch {
		switch msg.Topic {
		case "wallet":
			switch msg.Type {
			case "open":
				startWS()
			}
		case "ws":
			switch msg.Type {
			case "list":
				list := bus.B_WsList_Response{}
				connectionsMutex.Lock()
				for _, conn := range connections {
					list = append(list, bus.B_WsList_Conn{
						Agent: conn.Agent,
					})
				}
				connectionsMutex.Unlock()
				msg.Respond(list, nil)
			}
		}
	}
}

func startWS() {
	http.HandleFunc("/ws", web3Handler)

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(WEB3PRO_PORT),
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Hour,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Trace().Msgf("ws server started on port %d", WEB3PRO_PORT)
		err := server.ListenAndServe()
		if err != nil {
			log.Fatal().Err(err).Msgf("WS server failed to start on port %d", WEB3PRO_PORT)
		}
	}()
}

func AddConnection(conn *ConContext) {
	connectionsMutex.Lock()
	connections = append(connections, conn)
	connectionsMutex.Unlock()
}

func RemoveConnection(conn *ConContext) {
	connectionsMutex.Lock()
	for i, c := range connections {
		if c == conn {
			connections = append(connections[:i], connections[i+1:]...)
			break
		}
	}
	connectionsMutex.Unlock()
}

// Upgrade HTTP connection to a WebSocket connection
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin
		return true
	},
}

func web3Handler(w http.ResponseWriter, r *http.Request) {

	log.Debug().Msgf("Web3 handler called")

	if cmn.CurrentWallet == nil {
		http.Error(w, "Wallet not initialized", http.StatusInternalServerError)
		return
	}

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

	context := &ConContext{
		Agent: r.Header.Get("User-Agent"),
	}
	AddConnection(context)
	defer RemoveConnection(context)

	for {
		// Read message from browser
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Error().Msgf("Read error: %v", err)
			break
		}

		if msgType != websocket.TextMessage {
			log.Trace().Msgf("Received non-text message: %d", msgType)
			break
		}

		// Print the message to the console
		log.Trace().Msgf("Received: %s", msg)

		var rpcReq RPCRequest
		err = json.Unmarshal(msg, &rpcReq)
		if err != nil {
			log.Printf("JSON parse error: %v", err)
			continue
		}

		response := &RPCResponse{
			JSONRPC: "2.0",
			ID:      rpcReq.ID,
		}

		// Dispatch based on method prefix
		switch {
		case strings.HasPrefix(rpcReq.Method, "eth_"):
			handleEthMethod(rpcReq, context, response)
		case strings.HasPrefix(rpcReq.Method, "net_"):
			handleNetMethod(rpcReq, context, response)
		default:
			log.Printf("Unknown method: %s", rpcReq.Method)
			// Handle unknown methods or send an error response
			response.Error = map[string]interface{}{
				"code":    -32601,
				"message": "Method not found",
			}
		}
		sendResponse(conn, response)
	}
}

func sendResponse(conn *websocket.Conn, response *RPCResponse) {

	log.Debug().Msgf("Sending response: %v", response)

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
