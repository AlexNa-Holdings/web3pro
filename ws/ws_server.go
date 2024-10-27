package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
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
var blockedUA = map[string]bool{}

type RPCRequest struct {
	JSONRPC       string `json:"jsonrpc"`
	ID            int64  `json:"id"`
	Method        string `json:"method"`
	Params        any    `json:"params"` // Params as a slice of interface{}
	Web3ProOrigin string `json:"__web3proOrigin,omitempty"`
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

var start_once sync.Once

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
				start_once.Do(startWS)
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
	go func() {
		server := &http.Server{
			Addr:              ":" + strconv.Itoa(WEB3PRO_PORT),
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       2 * time.Hour,
			ReadHeaderTimeout: 5 * time.Second,
		}

		http.HandleFunc("/ws", web3Handler)

		for {
			err := server.ListenAndServe()
			if err != nil {
				log.Error().Err(err).Msgf("WS server failed to start on port %d", WEB3PRO_PORT)
				bus.Send("ui", "notify-error", "Failed to start WS server")
				time.Sleep(10 * time.Second) // Wait for 1 minute before retrying
				continue
			} else {
				break
			}
		}
		log.Trace().Msgf("ws server started on port %d", WEB3PRO_PORT)
		bus.Send("ui", "notify", "WS server started")
	}()
}

func broadcastChainChanged(data any) {
	u, ok := data.(string)
	if !ok {
		log.Error().Msgf("ws_broadcast: Invalid data type: %v", data)
		return
	}

	w := cmn.CurrentWallet
	if w == nil {
		log.Error().Msg("ws_broadcast: No wallet")
		return
	}

	o := w.GetOrigin(u)
	if o == nil {
		log.Error().Msgf("ws_broadcast: Origin not found: %s", u)
		return
	}

	for _, conn := range WSConnections {
		a := conn.SM.getSubsForEvent(u, "chainChanged")
		for _, sub := range a {
			conn.send(&RPCBroadcast{
				JSONRPC: "2.0",
				Method:  "chainChanged_subscription",
				Params: BroadcastParams{
					Subscription: sub.id,
					Result:       fmt.Sprintf("0x%x", o.ChainId),
				},
				Web3ProOrigin: u,
			})
		}
	}
}

func broadcastAddressesChanged(data any) {

	log.Debug().Msgf("broadcastAddressesChanged: %v", data)

	u, ok := data.(string)
	if !ok {
		log.Error().Msgf("ws_broadcast: Invalid data type: %v", data)
		return
	}

	w := cmn.CurrentWallet
	if w == nil {
		log.Error().Msg("ws_broadcast: No wallet")
		return
	}

	o := w.GetOrigin(u)
	if o == nil {
		log.Error().Msgf("ws_broadcast: Origin not found: %s", u)
		return
	}

	addrs := make([]string, 0)
	for _, a := range o.Addresses {
		addrs = append(addrs, a.String())
	}

	for _, conn := range WSConnections {
		a := conn.SM.getSubsForEvent(u, "accountsChanged")
		for _, sub := range a {
			conn.send(&RPCBroadcast{
				JSONRPC: "2.0",
				Method:  "accountsChanged_subscription",
				Params: BroadcastParams{
					Subscription: sub.id,
					Result:       addrs,
				},
				Web3ProOrigin: u,
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

func extractVersion(ua string, marker string) string {
	start := strings.Index(ua, marker)
	if start == -1 {
		return "Unknown"
	}

	start += len(marker)
	end := strings.IndexAny(ua[start:], " ;)")
	if end == -1 {
		end = len(ua)
	} else {
		end += start
	}

	return ua[start:end]
}

func getBrowser(ua string) (string, string) {

	log.Debug().Msgf("getBrowser: %s", ua)

	var browserName, fullVersion string

	if strings.Contains(ua, "OPR") {
		browserName = "Opera"
		fullVersion = extractVersion(ua, "OPR/")
	} else if strings.Contains(ua, "Edg") {
		browserName = "Edge"
		fullVersion = extractVersion(ua, "Edg/")
	} else if strings.Contains(ua, "Chrome") {
		browserName = "Chrome"
		fullVersion = extractVersion(ua, "Chrome/")
		if strings.Contains(ua, "Brave") || strings.Contains(ua, "brave") {
			browserName = "Brave"
		}
	} else if strings.Contains(ua, "Safari") && !strings.Contains(ua, "Chrome") {
		browserName = "Safari"
		fullVersion = extractVersion(ua, "Version/")
	} else if strings.Contains(ua, "Firefox") {
		browserName = "Firefox"
		fullVersion = extractVersion(ua, "Firefox/")
	} else {
		browserName = "Unknown"
		fullVersion = "Unknown"
	}

	return browserName, fullVersion
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

	allowed := false

	ua := r.Header.Get("User-Agent")

	if blockedUA[ua] {
		http.Error(w, "Connection not allowed", http.StatusForbidden)
		return
	}

	browser, version := getBrowser(ua)

	bus.FetchEx("ui", "hail", &bus.B_Hail{
		Title: "Allow Browser Connection",
		Template: `<c><w><blink>Allow</blink> connection from browser?

` + browser + `
version: ` + version + `

<button text:Ok> <button text:Cancel>

<button text:'Block for session' id:block>`,
		OnOk: func(m *bus.Message, v *gocui.View) bool {
			allowed = true
			return true
		},
		OnClickHotspot: func(m *bus.Message, v *gocui.View, hs *gocui.Hotspot) {
			if hs.Value == "button block" {
				allowed = false
				blockedUA[ua] = true
				bus.Send("ui", "remove-hail", m)
			}
		}},
		0, 1*time.Minute, 1*time.Minute, nil, 0)
	if !allowed {
		http.Error(w, "Connection not allowed", http.StatusForbidden)
		return
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

		log.Debug().Msgf("ws-> %v", string(msg))

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

		// Dispatch based on the method prefix

		if rpcReq.Method == "personal_sign" {
			rpcReq.Method = "eth_sign"
		}

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

	log.Debug().Msgf("ws<- %v", string(respBytes))

	err = con.Connection.WriteMessage(websocket.TextMessage, respBytes)
	if err != nil {
		log.Printf("Write error: %v", err)
	}
}

func getAllowedOrigin(u string) (*cmn.Origin, bool) {

	log.Debug().Msgf("getAllowedOrigin: %s", u)

	su, err := url.Parse(u)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to parse URL: %s", u)
		return nil, false
	}

	if su.Scheme != "https" && su.Scheme != "http" {
		log.Error().Msgf("Invalid scheme: %s", su.Scheme)
		return nil, false
	}

	w := cmn.CurrentWallet
	if w == nil {
		return nil, false
	}

	allowed := false
	origin := w.GetOrigin(u)
	if origin == nil {
		bus.Fetch("ui", "hail", &bus.B_Hail{
			Title: "Connect Web Application",
			Template: `<c><w>
Allow to connect to this web application:

<u><b>` + cmn.GetHostName(u) + `</b></u>

and use the current chain & address?

<button text:Ok> <button text:Cancel>`,
			OnOk: func(m *bus.Message, v *gocui.View) bool {

				chain_id := 1
				b := w.GetBlockchainByName(w.CurrentChain)
				if b != nil {
					chain_id = b.ChainId
				}

				origin = &cmn.Origin{
					URL:       u,
					ChainId:   chain_id,
					Addresses: []common.Address{w.CurrentAddress},
				}

				w.AddOrigin(origin)
				w.CurrentOrigin = u
				err := w.Save()
				if err != nil {
					log.Error().Err(err).Msg("Failed to save wallet")
					bus.Send("ui", "notify", "Failed to save wallet")
				}
				allowed = true
				return true
			}})
	} else {
		allowed = true
	}

	return origin, allowed
}
