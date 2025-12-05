package ui

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/eth"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

type TokenPane struct {
	PaneDescriptor
	On bool
}

type tokenInfo struct {
	Token        *cmn.Token
	TotalBalance *big.Int
	TotalUSD     float64
}

var token_info_list []*tokenInfo = make([]*tokenInfo, 0)

var Token TokenPane = TokenPane{
	PaneDescriptor: PaneDescriptor{
		MinWidth:               80,
		MinHeight:              2,
		MaxHeight:              30,
		SupportCachedHightCalc: true,
	},
}

func (p *TokenPane) IsOn() bool {
	return p.On
}

func (p *TokenPane) SetOn(on bool) {
	wasOff := !p.On
	p.On = on
	// Trigger update when pane is turned on
	if on && wasOff {
		go p.updateList()
	}
}

func (p *TokenPane) GetDesc() *PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *TokenPane) EstimateLines(w int) int {
	return gocui.EstimateTemplateLines(p.GetTemplate(), w)
}

func (p *TokenPane) SetView(x0, y0, x1, y1 int, overlap byte) {
	v, err := Gui.SetView("tokens", x0, y0, x1, y1, overlap)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}

		p.PaneDescriptor.View = v
		v.JoinedFrame = true
		v.Title = "Tokens"
		v.ScrollBar = true
		v.OnResize = func(v *gocui.View) {
			v.RenderTemplate(p.GetTemplate())
			v.ScrollTop()
		}
		v.OnOverHotspot = ProcessOnOverHotspot
		v.OnClickHotspot = ProcessOnClickHotspot

		p.SetTemplate(p.rebuildTemplate())
		v.RenderTemplate(p.GetTemplate())
	}
}

var tokenUpdateMu sync.Mutex
var tokenUpdatePending bool
var tokenLastUpdate time.Time

func TokenLoop() {
	ch := bus.Subscribe("wallet", "price", "eth")
	defer bus.Unsubscribe(ch)

	for msg := range ch {
		go Token.processToken(msg)
	}
}

func (p *TokenPane) processToken(msg *bus.Message) {
	switch msg.Topic {
	case "wallet":
		switch msg.Type {
		case "saved":
			p.scheduleUpdate()
		// Note: "open" is not handled here - we wait for eth "connected" events
		}
	case "price":
		switch msg.Type {
		case "updated":
			p.scheduleUpdate()
		}
	case "eth":
		switch msg.Type {
		case "connected":
			// A blockchain connection was established, schedule update with debounce
			p.scheduleUpdate()
		}
	}
}

// scheduleUpdate debounces updateList calls to avoid flooding RPC on startup
func (p *TokenPane) scheduleUpdate() {
	tokenUpdateMu.Lock()
	defer tokenUpdateMu.Unlock()

	// If update is already pending, skip
	if tokenUpdatePending {
		return
	}

	// Debounce: wait at least 2 seconds between updates
	timeSinceLastUpdate := time.Since(tokenLastUpdate)
	if timeSinceLastUpdate < 2*time.Second {
		tokenUpdatePending = true
		go func() {
			time.Sleep(2*time.Second - timeSinceLastUpdate)
			tokenUpdateMu.Lock()
			tokenUpdatePending = false
			tokenLastUpdate = time.Now()
			tokenUpdateMu.Unlock()
			p.updateList()
		}()
		return
	}

	tokenLastUpdate = time.Now()
	go p.updateList()
}

func (p *TokenPane) updateList() {
	if !p.On {
		return
	}

	w := cmn.CurrentWallet
	if w == nil {
		return
	}

	if len(w.Tokens) == 0 {
		return
	}

	// Group tokens by chain for batch processing
	tokensByChain := make(map[int][]*cmn.Token)
	for _, t := range w.Tokens {
		// Skip ignored tokens
		if t.Ignored {
			continue
		}
		tokensByChain[t.ChainId] = append(tokensByChain[t.ChainId], t)
	}

	// Map to store total balances per token
	tokenBalances := make(map[*cmn.Token]*big.Int)

	// Process each chain
	for chainId, tokens := range tokensByChain {
		b := w.GetBlockchain(chainId)
		if b == nil {
			continue
		}

		// Separate native and ERC20 tokens
		var nativeTokens []*cmn.Token
		var erc20Tokens []*cmn.Token
		for _, t := range tokens {
			if t.Native {
				nativeTokens = append(nativeTokens, t)
			} else {
				erc20Tokens = append(erc20Tokens, t)
			}
		}

		// Handle native tokens with multicall batch
		if len(nativeTokens) > 0 && len(w.Addresses) > 0 {
			// Build batch queries for all addresses
			queries := make([]*eth.NativeBalanceQuery, 0, len(w.Addresses))
			for _, addr := range w.Addresses {
				queries = append(queries, &eth.NativeBalanceQuery{
					Holder: addr.Address,
				})
			}

			// Execute batch query
			err := eth.GetNativeBalancesBatch(b, queries)
			if err != nil {
				log.Debug().Err(err).Msgf("Batch native balance query failed for chain %d", chainId)
			}

			// Sum up balances for all addresses
			totalBalance := big.NewInt(0)
			for _, q := range queries {
				if q.Balance != nil {
					totalBalance.Add(totalBalance, q.Balance)
				}
			}

			// Assign the same total to all native tokens on this chain (there's typically just one)
			for _, t := range nativeTokens {
				tokenBalances[t] = new(big.Int).Set(totalBalance)
			}
		}

		// Handle ERC20 tokens with multicall
		if len(erc20Tokens) > 0 && len(w.Addresses) > 0 {
			// Build batch queries for all (token, address) pairs
			queries := make([]*eth.BalanceQuery, 0, len(erc20Tokens)*len(w.Addresses))
			queryMap := make(map[*eth.BalanceQuery]*cmn.Token) // To map query back to token

			for _, t := range erc20Tokens {
				for _, addr := range w.Addresses {
					q := &eth.BalanceQuery{
						Token:  t,
						Holder: addr.Address,
					}
					queries = append(queries, q)
					queryMap[q] = t
				}
			}

			// Execute batch query
			err := eth.GetERC20BalancesBatch(b, queries)
			if err != nil {
				log.Debug().Err(err).Msgf("Batch balance query failed for chain %d", chainId)
			}

			// Aggregate balances per token
			for _, q := range queries {
				t := queryMap[q]
				if tokenBalances[t] == nil {
					tokenBalances[t] = big.NewInt(0)
				}
				if q.Balance != nil {
					tokenBalances[t].Add(tokenBalances[t], q.Balance)
				}
			}
		}
	}

	// Build the list
	totalUSD := 0.0
	list := make([]*tokenInfo, 0)

	for t, totalBalance := range tokenBalances {
		if totalBalance == nil || totalBalance.Cmp(big.NewInt(0)) == 0 {
			continue
		}

		usdValue := t.Price * t.Float64(totalBalance)

		// Skip tokens below minimum value threshold
		if usdValue < cmn.Config.MinTokenValue {
			continue
		}

		totalUSD += usdValue

		list = append(list, &tokenInfo{
			Token:        t,
			TotalBalance: totalBalance,
			TotalUSD:     usdValue,
		})
	}

	// Sort by USD value descending
	sort.Slice(list, func(i, j int) bool {
		return list[i].TotalUSD > list[j].TotalUSD
	})

	token_info_list = list

	if Token.View != nil {
		Gui.Update(func(g *gocui.Gui) error {
			if Token.View != nil {
				p.SetTemplate(Token.rebuildTemplate())
				Token.View.RenderTemplate(p.GetTemplate())
				Token.View.ScrollTop()
			}
			return nil
		})
	}

	if Token.View != nil {
		Token.View.Subtitle = fmt.Sprintf("N:%d %s", len(list),
			cmn.FmtFloat64D(totalUSD, true))
	}
}

func (p *TokenPane) rebuildTemplate() string {
	w := cmn.CurrentWallet
	if w == nil {
		return "no open wallet"
	}

	if len(w.Tokens) == 0 {
		return "(no tokens)"
	}

	if len(token_info_list) == 0 {
		return "loading..."
	}

	temp := "Symbol   Chain Price   Change   Balance       Value\n"

	for i, ti := range token_info_list {
		t := ti.Token

		b := w.GetBlockchain(t.ChainId)
		if b == nil {
			continue
		}

		temp += fmt.Sprintf("%-8s ", t.Symbol)

		temp += fmt.Sprintf("%-5s ", b.GetShortName())

		if t.Price > 0 {
			temp += cmn.TagDollarLink(t.Price)
		} else {
			temp += "          "
		}

		if t.PriceChange24 > 0 {
			temp += fmt.Sprintf("<color fg:green>\uf0d8%5.2f%%</color> ", t.PriceChange24)
		} else if t.PriceChange24 < 0 {
			temp += fmt.Sprintf("<color fg:red>\uf0d7%5.2f%%</color> ", -t.PriceChange24)
		} else {
			temp += "         "
		}

		temp += cmn.TagValueLink(ti.TotalBalance, t)

		temp += cmn.TagDollarLink(ti.TotalUSD)

		if i < len(token_info_list)-1 {
			temp += "\n"
		}
	}
	return temp
}
