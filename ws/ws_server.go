package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const WEB3PRO_PORT = 9323 // WEB3 mnemonics

type ConContext struct {
	Agent      string
	Connection *websocket.Conn
	SM         *subManger
}

var WSConnections = make([]*ConContext, 0)
var WSConnectionsMutex = sync.Mutex{}

type RPCRequest struct {
	JSONRPC       string        `json:"jsonrpc"`
	ID            int64         `json:"id"`
	Method        string        `json:"method"`
	Params        []interface{} `json:"params"` // Params as a slice of interface{}
	Web3ProOrigin string        `json:"__web3proOrigin,omitempty"`
}
type BroadcastParams struct {
	Subscription string `json:"subscription,omitempty"`
	Result       any    `json:"result,omitempty"`
}

type RPCBroadcast struct {
	JSONRPC       string          `json:"jsonrpc"`
	Method        string          `json:"method"`
	Params        BroadcastParams `json:"params"`
	Web3ProOrigin string          `json:"__web3proOrigin,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Result  interface{} `json:"result"`
	Error   *RPCError   `json:"error,omitempty"`
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
			case "origin-chain-changed":
				broadcastChainChanged(msg.Data)
			case "origin-addresses-changed":
				broadcastAddressesChanged(msg.Data)
			case "origin-changed":
				broadcastChainChanged(msg.Data)
				broadcastAddressesChanged(msg.Data)
			}
		case "ws":
			switch msg.Type {
			case "list":
				list := bus.B_WsList_Response{}
				WSConnectionsMutex.Lock()
				for _, conn := range WSConnections {
					list = append(list, bus.B_WsList_Conn{
						Agent: conn.Agent,
					})
				}
				WSConnectionsMutex.Unlock()
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

func broadcastChainChanged(data any) {
	url, ok := data.(string)
	if !ok {
		log.Error().Msgf("ws_broadcast: Invalid data type: %v", data)
		return
	}

	w := cmn.CurrentWallet
	if w == nil {
		log.Error().Msg("ws_broadcast: No wallet")
		return
	}

	o := w.GetOrigin(url)
	if o == nil {
		log.Error().Msgf("ws_broadcast: Origin not found: %s", url)
		return
	}

	for _, conn := range WSConnections {
		a := conn.SM.getSubsForEvent(url, "chainChanged")
		for _, sub := range a {
			conn.send(&RPCBroadcast{
				JSONRPC: "2.0",
				Method:  "chainChanged_subscription",
				Params: BroadcastParams{
					Subscription: sub.id,
					Result:       fmt.Sprintf("0x%x", o.ChainId),
				},
				Web3ProOrigin: url,
			})
		}
	}
}

func broadcastAddressesChanged(data any) {

	log.Debug().Msgf("broadcastAddressesChanged: %v", data)

	url, ok := data.(string)
	if !ok {
		log.Error().Msgf("ws_broadcast: Invalid data type: %v", data)
		return
	}

	w := cmn.CurrentWallet
	if w == nil {
		log.Error().Msg("ws_broadcast: No wallet")
		return
	}

	o := w.GetOrigin(url)
	if o == nil {
		log.Error().Msgf("ws_broadcast: Origin not found: %s", url)
		return
	}

	addrs := make([]string, 0)
	for _, a := range o.Addresses {
		addrs = append(addrs, a.String())
	}

	for _, conn := range WSConnections {
		a := conn.SM.getSubsForEvent(url, "accountsChanged")
		for _, sub := range a {
			conn.send(&RPCBroadcast{
				JSONRPC: "2.0",
				Method:  "accountsChanged_subscription",
				Params: BroadcastParams{
					Subscription: sub.id,
					Result:       addrs,
				},
				Web3ProOrigin: url,
			})
		}
	}
}

func AddConnection(conn *ConContext) {
	WSConnectionsMutex.Lock()
	WSConnections = append(WSConnections, conn)
	WSConnectionsMutex.Unlock()
}

func RemoveConnection(conn *ConContext) {
	WSConnectionsMutex.Lock()
	for i, c := range WSConnections {
		if c == conn {
			WSConnections = append(WSConnections[:i], WSConnections[i+1:]...)
			break
		}
	}
	WSConnectionsMutex.Unlock()
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

	// Upgrade HTTP connection to a WebSocket connection
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			log.Debug().Msgf("CheckOrigin: %s", r.Header.Get("Origin"))
			// Allow WSConnections from any origin
			return true
		},
	}
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error()
		return
	}
	defer conn.Close()

	ctx := &ConContext{
		Agent:      r.Header.Get("User-Agent"),
		Connection: conn,
		SM:         newSubManager(),
	}
	AddConnection(ctx)
	defer RemoveConnection(ctx)

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

		log.Printf("ws->%v", rpcReq)

		// Dispatch based on method prefix
		switch {
		case strings.HasPrefix(rpcReq.Method, "eth_"):
			handleEthMethod(rpcReq, ctx, response)
		case strings.HasPrefix(rpcReq.Method, "net_"):
			handleNetMethod(rpcReq, ctx, response)
		case strings.HasPrefix(rpcReq.Method, "wallet_"):
			handleWalletMethod(rpcReq, ctx, response)
		default:
			log.Printf("Unknown method: %v", rpcReq.Method)
			// Handle unknown methods or send an error response
			response.Error = &RPCError{
				Code:    -32601,
				Message: "Method not found",
			}
		}

		ctx.send(response)
	}
}

func (con *ConContext) send(data any) {
	respBytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return
	}

	log.Debug().Msgf("Sending response: %v", string(respBytes))

	err = con.Connection.WriteMessage(websocket.TextMessage, respBytes)
	if err != nil {
		log.Printf("Write error: %v", err)
	}
}

func getAllowedOrigin(url string) (*cmn.Origin, bool) {

	log.Debug().Msgf("getAllowedOrigin: %s", url)

	w := cmn.CurrentWallet
	if w == nil {
		return nil, false
	}

	allowed := false
	origin := w.GetOrigin(url)
	if origin == nil {
		bus.Fetch("ui", "hail", &bus.B_Hail{
			Title: "Connect Web Application",
			Template: `<c><w>
Allow to connect to this web application:

<u><b>` + url + `</b></u>

and use the current chain & address?

<button text:Ok> <button text:Cancel>`,
			OnOk: func(m *bus.Message) {

				chain_id := 1
				b := w.GetBlockchain(w.CurrentChain)
				if b != nil {
					chain_id = b.ChainId
				}

				origin = &cmn.Origin{
					URL:       url,
					ChainId:   chain_id,
					Addresses: []common.Address{w.CurrentAddress},
				}

				w.AddOrigin(origin)
				err := w.Save()
				if err != nil {
					log.Error().Err(err).Msg("Failed to save wallet")
					bus.Send("ui", "notify", "Failed to save wallet")
				}
				allowed = true
				bus.Send("ui", "remove-hail", m)
			}})
	} else {
		allowed = true
	}

	return origin, allowed
}
