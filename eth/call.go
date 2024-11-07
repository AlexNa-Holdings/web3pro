package eth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum"
	"github.com/rs/zerolog/log"
)

// anti flood cache
// we will reuse the data if the same call is made within N seconds
const TIME_IN_CACHE = time.Duration(12) * time.Second
const CACHE_CLEANUP_INTERVAL = time.Duration(120) * time.Second

var lastCleanup time.Time

type CacheItem struct {
	data      []byte
	timestamp time.Time
}

type Cache struct {
	mu    sync.Mutex
	store map[string]CacheItem
}

var cache = Cache{
	mu:    sync.Mutex{},
	store: make(map[string]CacheItem),
}

// Generates a unique key for the CallMsg using hashing
func generateCacheKey(chain_id int, callMsg ethereum.CallMsg) string {
	hash := sha256.New()
	hash.Write([]byte(fmt.Sprintf("%d", chain_id)))
	hash.Write(callMsg.To.Bytes())
	hash.Write(callMsg.From.Bytes())
	if callMsg.Value != nil {
		hash.Write(callMsg.Value.Bytes())
	}
	if callMsg.Data != nil {
		hash.Write(callMsg.Data)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// Get cached data if available and not expired
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.store[key]
	if !exists {
		return nil, false
	}

	// Check if the cached item is older than 12 seconds
	if time.Since(item.timestamp) > TIME_IN_CACHE {
		delete(c.store, key) // Clean up expired cache
		return nil, false
	}

	return item.data, true
}

// Set data in cache
func (c *Cache) Set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Since(lastCleanup) > CACHE_CLEANUP_INTERVAL {
		// Clean up expired cache
		for key, item := range c.store {
			if time.Since(item.timestamp) > TIME_IN_CACHE {
				delete(c.store, key)
			}
		}
		lastCleanup = time.Now()
	}

	c.store[key] = CacheItem{
		data:      data,
		timestamp: time.Now(),
	}
}

func call(msg *bus.Message) (string, error) {
	req, ok := msg.Data.(*bus.B_EthCall)
	if !ok {
		return "", fmt.Errorf("invalid tx: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return "", errors.New("no wallet")
	}

	b := w.GetBlockchain(req.ChainId)
	if b == nil {
		return "", fmt.Errorf("call: blockchain not found: %v", req.ChainId)
	}

	c, ok := cons[b.ChainId]
	if !ok {
		log.Error().Msgf("SendSignedTx: Client not found for chainId: %d", b.ChainId)
		return "", fmt.Errorf("client not found for chainId: %d", b.ChainId)
	}

	call_msg := ethereum.CallMsg{
		To:    &req.To,
		From:  req.From,
		Value: req.Amount,
		Data:  req.Data,
	}

	// Check if the data is available in cache
	key := generateCacheKey(b.ChainId, call_msg)
	if data, exists := cache.Get(key); exists {
		return fmt.Sprintf("0x%x", data), nil
	} else {
		output, err := c.CallContract(context.Background(), call_msg, nil)
		if err != nil {
			log.Error().Msgf("call: Cannot call contract. Error:(%v)", err)
			return "", err
		}

		cache.Set(key, output)
		return fmt.Sprintf("0x%x", output), nil
	}
}
