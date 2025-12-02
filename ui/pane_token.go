package ui

import (
	"errors"
	"fmt"
	"math/big"
	"sort"

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

func TokenLoop() {
	ch := bus.Subscribe("wallet", "price")
	defer bus.Unsubscribe(ch)

	for msg := range ch {
		go Token.processToken(msg)
	}
}

func (p *TokenPane) processToken(msg *bus.Message) {
	switch msg.Topic {
	case "wallet":
		switch msg.Type {
		case "open", "saved":
			p.updateList()
		}
	case "price":
		switch msg.Type {
		case "updated":
			p.updateList()
		}
	}
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

		// Handle native tokens (need individual balance calls)
		for _, t := range nativeTokens {
			totalBalance := big.NewInt(0)
			for _, addr := range w.Addresses {
				balance, err := eth.GetBalance(b, addr.Address)
				if err != nil {
					log.Debug().Err(err).Msgf("Error getting native balance for %s", t.Symbol)
					continue
				}
				totalBalance.Add(totalBalance, balance)
			}
			tokenBalances[t] = totalBalance
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

	temp := "Symbol   Chain        Price   Change   Balance       Value\n"

	for i, ti := range token_info_list {
		t := ti.Token

		b := w.GetBlockchain(t.ChainId)
		if b == nil {
			continue
		}

		temp += fmt.Sprintf("%-8s ", t.Symbol)

		// Truncate chain name with ellipsis if too long
		chainName := b.Name
		if len(chainName) > 12 {
			chainName = chainName[:9] + "..."
		}
		temp += fmt.Sprintf("%-12s ", chainName)

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
