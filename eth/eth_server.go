package eth

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
)

type con struct {
	*ethclient.Client
	URL string
}

var cons map[int]*con = make(map[int]*con) // chainId -> client
var consMutex = sync.Mutex{}

// Rate limiter per chain with auto-tuning
type rateLimiter struct {
	chainId       int
	tokens        int
	maxTokens     int
	lastFill      time.Time
	lastSuccess   time.Time // last successful call without 429
	backoffUntil  time.Time // don't make any calls until this time
	autoMode      bool      // true = auto-tune rate, false = fixed rate
	lastCallTime  time.Time // when last RPC call was made
	mu            sync.Mutex
}

const (
	initialRateLimit = 5   // start with 5 calls/sec (conservative for cold start)
	minRateLimit     = 1   // minimum 1 call/sec
	maxRateLimit     = 100 // maximum 100 calls/sec
	increaseInterval = 60 * time.Second // increase rate after 60 seconds without errors
	increasePercent  = 10  // increase by 10%
	decreasePercent  = 50  // decrease by 50% on 429 error
)

var rateLimiters = make(map[int]*rateLimiter) // chainId -> rate limiter
var rateLimitersMu sync.Mutex

// getRateLimiter returns or creates a rate limiter for the given chain
func getRateLimiter(chainId int) *rateLimiter {
	rateLimitersMu.Lock()
	defer rateLimitersMu.Unlock()

	if rl, ok := rateLimiters[chainId]; ok {
		return rl
	}

	// Get settings from blockchain - default is auto mode with initialRateLimit
	startRate := initialRateLimit
	autoMode := true // default to auto mode
	w := cmn.CurrentWallet
	if w != nil {
		b := w.GetBlockchain(chainId)
		if b != nil {
			autoMode = b.RPCRateAuto || b.RPCRateLimit == 0 // auto if explicitly set or if rate is 0/not set
			if b.RPCRateLimit > 0 {
				startRate = b.RPCRateLimit
			}
			log.Debug().Int("chainId", chainId).Int("rate", startRate).Bool("auto", autoMode).Msg("RPC rate limiter initialized")
		}
	}

	rl := &rateLimiter{
		chainId:      chainId,
		tokens:       startRate,
		maxTokens:    startRate,
		lastFill:     time.Now(),
		lastSuccess:  time.Now(),
		lastCallTime: time.Now(), // Initialize to now so first call goes through, then rate limiting kicks in
		autoMode:     autoMode,
	}
	rateLimiters[chainId] = rl
	return rl
}

// waitForToken blocks until a token is available (rate limiting)
// This ensures strict rate limiting: at N calls/sec, there is at least 1/N seconds between calls
func (rl *rateLimiter) waitForToken() {
	for {
		rl.mu.Lock()

		now := time.Now()

		// If in backoff period, wait until it's over
		if now.Before(rl.backoffUntil) {
			waitTime := time.Until(rl.backoffUntil)
			rl.mu.Unlock()
			time.Sleep(waitTime)
			continue // Re-check everything after backoff
		}

		// Calculate minimum time between calls based on rate
		minInterval := time.Second / time.Duration(rl.maxTokens)
		timeSinceLastCall := now.Sub(rl.lastCallTime)

		// If not enough time has passed since last call, wait
		if timeSinceLastCall < minInterval {
			waitTime := minInterval - timeSinceLastCall
			rl.mu.Unlock()
			time.Sleep(waitTime)
			continue // Re-check everything after waiting
		}

		// Check if we should increase rate (no errors for a while) - only in auto mode
		if rl.autoMode && time.Since(rl.lastSuccess) > increaseInterval && rl.maxTokens < maxRateLimit {
			newMax := rl.maxTokens + (rl.maxTokens * increasePercent / 100)
			if newMax > maxRateLimit {
				newMax = maxRateLimit
			}
			if newMax > rl.maxTokens {
				oldMax := rl.maxTokens
				rl.maxTokens = newMax
				log.Debug().Int("chainId", rl.chainId).Int("oldRate", oldMax).Int("newRate", newMax).Msg("Rate limit increased (auto)")
				go persistRateLimit(rl.chainId, newMax) // persist in background
			}
			rl.lastSuccess = time.Now()
		}

		// Mark this call time and release lock
		prevCallTime := rl.lastCallTime
		callTime := time.Now()
		rl.lastCallTime = callTime
		actualInterval := callTime.Sub(prevCallTime)
		rl.mu.Unlock()

		// Log exact RPC call time for debugging
		log.Trace().Int("chainId", rl.chainId).Int("rate", rl.maxTokens).
			Str("time", callTime.Format("15:04:05.000")).
			Str("interval", actualInterval.String()).
			Msg("RPC call acquired")
		return
	}
}

// onRateLimitError is called when a 429 error is received - reduces rate and sets backoff
func (rl *rateLimiter) onRateLimitError() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	errorTime := time.Now()
	timeSinceLastCall := errorTime.Sub(rl.lastCallTime)

	// Log the 429 error with exact timing
	log.Warn().Int("chainId", rl.chainId).Int("rate", rl.maxTokens).
		Str("time", errorTime.Format("15:04:05.000")).
		Str("sinceLast", timeSinceLastCall.String()).
		Msg("429 rate limit error")

	// Set backoff period - wait 5 seconds before making any more calls
	// This gives the RPC provider time to reset their rate limit counters
	rl.backoffUntil = time.Now().Add(5 * time.Second)
	rl.tokens = 0 // Reset tokens to enforce immediate slowdown

	// Only auto-decrease rate in auto mode
	if rl.autoMode {
		newMax := rl.maxTokens - (rl.maxTokens * decreasePercent / 100)
		if newMax < minRateLimit {
			newMax = minRateLimit
		}
		if newMax < rl.maxTokens {
			log.Debug().Int("chainId", rl.chainId).Int("oldRate", rl.maxTokens).Int("newRate", newMax).Msg("Rate limit decreased due to 429 error (auto)")
			rl.maxTokens = newMax
			go persistRateLimit(rl.chainId, newMax) // persist in background
		}
	}
}

// onSuccess is called when a call succeeds - updates lastSuccess time
func (rl *rateLimiter) onSuccess() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.lastSuccess = time.Now()
}

// persistRateLimit saves the rate limit to the blockchain settings (only in auto mode)
func persistRateLimit(chainId int, rate int) {
	w := cmn.CurrentWallet
	if w == nil {
		return
	}
	b := w.GetBlockchain(chainId)
	if b == nil {
		return
	}
	// Only persist if in auto mode
	if b.RPCRateAuto || b.RPCRateLimit == 0 {
		if b.RPCRateLimit != rate {
			b.RPCRateLimit = rate
			w.Save()
		}
	}
}

// GetCurrentRateLimit returns the current rate limit for a chain (for display purposes)
func GetCurrentRateLimit(chainId int) int {
	rateLimitersMu.Lock()
	defer rateLimitersMu.Unlock()

	if rl, ok := rateLimiters[chainId]; ok {
		rl.mu.Lock()
		defer rl.mu.Unlock()
		return rl.maxTokens
	}
	return initialRateLimit
}

// UpdateRateLimiterSettings updates an existing rate limiter with new settings from blockchain config
// This preserves timing state (lastCallTime, backoffUntil) to avoid race conditions
func UpdateRateLimiterSettings(chainId int) {
	rateLimitersMu.Lock()
	defer rateLimitersMu.Unlock()

	w := cmn.CurrentWallet
	if w == nil {
		return
	}
	b := w.GetBlockchain(chainId)
	if b == nil {
		return
	}

	newAutoMode := b.RPCRateAuto || b.RPCRateLimit == 0
	newRate := initialRateLimit
	if b.RPCRateLimit > 0 {
		newRate = b.RPCRateLimit
	}

	rl, exists := rateLimiters[chainId]
	if !exists {
		// No existing rate limiter, will be created on next use
		return
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Only update if settings actually changed
	if rl.autoMode != newAutoMode || rl.maxTokens != newRate {
		log.Debug().Int("chainId", chainId).
			Int("oldRate", rl.maxTokens).Int("newRate", newRate).
			Bool("oldAuto", rl.autoMode).Bool("newAuto", newAutoMode).
			Msg("Rate limiter settings updated")
		rl.autoMode = newAutoMode
		rl.maxTokens = newRate
		rl.tokens = newRate
	}
}

// acquireRateLimit waits for rate limit token for the given chain
func acquireRateLimit(chainId int) {
	rl := getRateLimiter(chainId)
	rl.waitForToken()
}

// ReportRateLimitError should be called when a 429 error is detected
func ReportRateLimitError(chainId int) {
	rl := getRateLimiter(chainId)
	rl.onRateLimitError()
}

// ReportSuccess should be called after a successful RPC call
func ReportSuccess(chainId int) {
	rl := getRateLimiter(chainId)
	rl.onSuccess()
}

// isRateLimitError checks if an error is a 429 rate limit error
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "429") || strings.Contains(errStr, "Too Many Requests") || strings.Contains(errStr, "rate limit")
}

// isGatewayError checks if an error is a 502/503/504 gateway error
func isGatewayError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "502") || strings.Contains(errStr, "503") || strings.Contains(errStr, "504") ||
		strings.Contains(errStr, "Gateway") || strings.Contains(errStr, "Bad Gateway") ||
		strings.Contains(errStr, "Service Unavailable")
}

// getChainShortName returns the short name for a chain ID
func getChainShortName(chainId int) string {
	w := cmn.CurrentWallet
	if w != nil {
		b := w.GetBlockchain(chainId)
		if b != nil {
			return b.GetShortName()
		}
	}
	return fmt.Sprintf("Chain %d", chainId)
}

// handleRPCResult checks for rate limit errors and reports success/failure
func handleRPCResult(chainId int, err error) {
	if err != nil && isRateLimitError(err) {
		ReportRateLimitError(chainId)
		bus.Send("ui", "notify-error", fmt.Sprintf("%s: RPC rate limit (429)", getChainShortName(chainId)))
	} else if err != nil && isGatewayError(err) {
		bus.Send("ui", "notify-error", fmt.Sprintf("%s: RPC gateway error (504)", getChainShortName(chainId)))
	} else if err == nil {
		ReportSuccess(chainId)
	}
}

func Init() {
	LoadABIs()
	go Loop()
}

func Loop() {
	ch := bus.Subscribe("eth", "wallet")
	for msg := range ch {
		if msg.RespondTo != 0 {
			continue // ignore responses
		}
		go process(msg)
	}
}

func process(msg *bus.Message) {
	switch msg.Topic {
	case "eth":
		// Apply rate limiting for RPC calls
		chainId := getChainIdFromMessage(msg)
		if chainId > 0 {
			acquireRateLimit(chainId)
		}

		switch msg.Type {
		case "send":
			err := send(msg)
			handleRPCResult(chainId, err)
			msg.Respond(nil, err)
		case "send-tx":
			hash, err := sendTx(msg)
			handleRPCResult(chainId, err)
			msg.Respond(hash, err)
		case "call":
			data, err := call(msg)
			handleRPCResult(chainId, err)
			msg.Respond(data, err)
		case "multi-call":
			data, err := multiCall(msg)
			handleRPCResult(chainId, err)
			msg.Respond(data, err)
		case "sign-typed-data-v4":
			sig, err := signTypedDataV4(msg)
			msg.Respond(sig, err)
		case "sign":
			sig, err := sign(msg)
			msg.Respond(sig, err)
		case "estimate-gas":
			gas, err := estimateGas(msg)
			handleRPCResult(chainId, err)
			msg.Respond(gas, err)
		case "block-number":
			blockNumber, err := blockNumber(msg)
			handleRPCResult(chainId, err)
			msg.Respond(blockNumber, err)
		case "get-tx-by-hash":
			tx, err := getTxByHash(msg)
			handleRPCResult(chainId, err)
			msg.Respond(tx, err)
		}
	case "wallet":
		switch msg.Type {
		case "open":
			initConnections()
		case "saved":
			updateConnections()
		}
	}
}

// getChainIdFromMessage extracts chainId from various eth message types
func getChainIdFromMessage(msg *bus.Message) int {
	switch req := msg.Data.(type) {
	case *bus.B_EthCall:
		return req.ChainId
	case *bus.B_EthMultiCall:
		return req.ChainId
	case *bus.B_EthSend:
		return req.ChainId
	case *bus.B_EthSendTx:
		return req.ChainId
	case *bus.B_EthEstimateGas:
		w := cmn.CurrentWallet
		if w != nil {
			b := w.GetBlockchainByName(req.Blockchain)
			if b != nil {
				return b.ChainId
			}
		}
	case *bus.B_EthBlockNumber:
		w := cmn.CurrentWallet
		if w != nil {
			b := w.GetBlockchainByName(req.Blockchain)
			if b != nil {
				return b.ChainId
			}
		}
	case *bus.B_EthTxByHash:
		w := cmn.CurrentWallet
		if w != nil {
			b := w.GetBlockchainByName(req.Blockchain)
			if b != nil {
				return b.ChainId
			}
		}
	}
	return 0
}

func initConnections() {
	consMutex.Lock()
	defer consMutex.Unlock()

	for _, c := range cons {
		c.Close()
	}
	cons = make(map[int]*con)

	w := cmn.CurrentWallet
	if w == nil {
		return
	}

	for _, b := range w.Blockchains {
		if _, ok := cons[b.ChainId]; !ok {
			openClient_locked(b)
		}
	}
}

func updateConnections() {
	consMutex.Lock()
	defer consMutex.Unlock()

	w := cmn.CurrentWallet
	if w == nil {
		return
	}

	vetted := make(map[int]bool)
	for _, b := range w.Blockchains {
		vetted[b.ChainId] = true

		c, ok := cons[b.ChainId]
		if !ok {
			openClient_locked(b)
		} else {

			if c.URL != b.Url {
				//reconnect
				cons[b.ChainId].Close()
				bus.Send("eth", "disconnected", b.ChainId)
				openClient_locked(b)
			}
		}

		// Update rate limiter settings (preserves timing state to avoid race conditions)
		UpdateRateLimiterSettings(b.ChainId)
	}

	to_delete := []int{}
	for c := range cons {
		if _, ok := vetted[c]; !ok {
			to_delete = append(to_delete, c)
		}
	}

	for _, c := range to_delete {
		cons[c].Close()
		delete(cons, c)
	}

}

func openClient_locked(b *cmn.Blockchain) error {
	client, err := ethclient.Dial(b.Url)
	if err != nil {
		log.Error().Err(err).Str("chain", b.GetShortName()).Str("url", b.Url).Msg("OpenClient: Cannot dial")
		return err
	}
	cons[b.ChainId] = &con{client, b.Url}
	log.Trace().Str("chain", b.GetShortName()).Msg("OpenClient: Client opened")
	bus.Send("eth", "connected", b.ChainId)
	return nil
}

func getEthClient(b *cmn.Blockchain) (*ethclient.Client, error) {
	consMutex.Lock()
	defer consMutex.Unlock()

	c, ok := cons[b.ChainId]
	if !ok {
		return nil, fmt.Errorf("client not found for chain %s", b.GetShortName())
	}

	if c.Client == nil {
		return nil, fmt.Errorf("client is nil for chain %s", b.GetShortName())
	}

	return c.Client, nil
}

func blockNumber(msg *bus.Message) (string, error) {
	req, ok := msg.Data.(*bus.B_EthBlockNumber)
	if !ok {
		return "", fmt.Errorf("invalid tx: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return "", fmt.Errorf("no wallet")
	}

	b := w.GetBlockchainByName(req.Blockchain)
	if b == nil {
		return "", fmt.Errorf("blockchain not found: %v", req.Blockchain)
	}

	c, err := getEthClient(b)
	if err != nil {
		return "", err
	}

	blockNumber, err := c.BlockNumber(context.Background())
	if err != nil {
		return "", err
	}

	n_hex := fmt.Sprintf("0x%x", blockNumber)

	return n_hex, nil
}

func getTxByHash(msg *bus.Message) (*bus.B_EthTxByHash_Response, error) {
	req, ok := msg.Data.(*bus.B_EthTxByHash)
	if !ok {
		return nil, fmt.Errorf("invalid tx: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("no wallet")
	}

	b := w.GetBlockchainByName(req.Blockchain)
	if b == nil {
		return nil, fmt.Errorf("blockchain not found: %v", req.Blockchain)
	}

	c, err := getEthClient(b)
	if err != nil {
		return nil, err
	}

	tx, pending, err := c.TransactionByHash(context.Background(), req.Hash)
	if err != nil {
		return nil, err
	}

	// Get block details if the transaction is confirmed
	var blockHash common.Hash
	var blockNumber *big.Int
	if !pending {
		txReceipt, err := c.TransactionReceipt(context.Background(), req.Hash)
		if err != nil {
			return nil, err
		}
		blockHash = txReceipt.BlockHash
		blockNumber = txReceipt.BlockNumber
	} else {
		blockHash = common.Hash{} // Placeholder for pending transactions
		blockNumber = big.NewInt(0)
	}

	from, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender: %v", err)
	}

	to := ""
	if tx.To() != nil {
		to = tx.To().Hex()
	}

	r, s, v := tx.RawSignatureValues()

	resp := bus.B_EthTxByHash_Response{
		BlockHash:        blockHash.Hex(),
		BlockNumber:      blockNumber.String(),
		ChainID:          b.ChainId,
		From:             from.Hex(),
		Gas:              fmt.Sprintf("%d", tx.Gas()),
		GasPrice:         tx.GasPrice().String(),
		Hash:             tx.Hash().Hex(),
		Input:            common.Bytes2Hex(tx.Data()),
		Nonce:            fmt.Sprintf("%d", tx.Nonce()),
		To:               to,
		TransactionIndex: "", // Needs to be filled if available
		Value:            tx.Value().String(),
		V:                v.String(),
		R:                r.String(),
		S:                s.String(),
	}

	if !pending {
		// Fill in the transaction index if the transaction is mined
		txReceipt, err := c.TransactionReceipt(context.Background(), req.Hash)
		if err != nil {
			return nil, err
		}
		resp.TransactionIndex = fmt.Sprintf("%d", txReceipt.TransactionIndex)
	}

	return &resp, nil
}
