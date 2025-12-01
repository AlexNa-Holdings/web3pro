package lp_v2

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
)

func getPositionStatus(msg *bus.Message) (*bus.B_LP_V2_GetPositionStatus_Response, error) {
	req, ok := msg.Data.(*bus.B_LP_V2_GetPositionStatus)
	if !ok {
		return nil, fmt.Errorf("invalid request: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return nil, fmt.Errorf("no wallet")
	}

	pos := w.GetLP_V2Position(req.ChainId, req.Factory, req.Pair)
	if pos == nil {
		return nil, fmt.Errorf("position not found")
	}

	lp := w.GetLP_V2(req.ChainId, req.Factory)
	if lp == nil {
		return nil, fmt.Errorf("provider not found")
	}

	b := w.GetBlockchain(pos.ChainId)
	if b == nil {
		return nil, fmt.Errorf("blockchain not found")
	}

	// Get token0 and token1 from the pair contract
	token0, err := getToken0(req.ChainId, req.Pair)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get token0")
		token0 = pos.Token0
	}

	token1, err := getToken1(req.ChainId, req.Pair)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get token1")
		token1 = pos.Token1
	}

	// Get LP balance for the owner
	lpBalance, err := getBalanceOf(req.ChainId, req.Pair, pos.Owner)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get LP balance")
		lpBalance = big.NewInt(0)
	}

	// Get total supply
	totalSupply, err := getTotalSupply(req.ChainId, req.Pair)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get total supply")
		totalSupply = big.NewInt(0)
	}

	// Get reserves
	reserve0 := big.NewInt(0)
	reserve1 := big.NewInt(0)
	reservesResp := msg.Fetch("lp_v2", "get-reserves", &bus.B_LP_V2_GetReserves{
		ChainId: req.ChainId,
		Pair:    req.Pair,
	})
	if reservesResp.Error == nil {
		if reserves, ok := reservesResp.Data.(*bus.B_LP_V2_GetReserves_Response); ok {
			reserve0 = reserves.Reserve0
			reserve1 = reserves.Reserve1
		}
	}

	// Calculate user's share of liquidity
	liquidity0 := big.NewInt(0)
	liquidity1 := big.NewInt(0)

	if totalSupply.Cmp(big.NewInt(0)) > 0 && lpBalance.Cmp(big.NewInt(0)) > 0 {
		// liquidity0 = reserve0 * lpBalance / totalSupply
		liquidity0 = new(big.Int).Mul(reserve0, lpBalance)
		liquidity0.Div(liquidity0, totalSupply)

		// liquidity1 = reserve1 * lpBalance / totalSupply
		liquidity1 = new(big.Int).Mul(reserve1, lpBalance)
		liquidity1.Div(liquidity1, totalSupply)
	}

	// Calculate dollar values
	liquidity0Dollars := 0.0
	liquidity1Dollars := 0.0
	t0 := w.GetTokenByAddress(pos.ChainId, token0)
	t1 := w.GetTokenByAddress(pos.ChainId, token1)

	if t0 != nil {
		liquidity0Dollars = t0.Float64(liquidity0) * t0.Price
	}
	if t1 != nil {
		liquidity1Dollars = t1.Float64(liquidity1) * t1.Price
	}

	pn := fmt.Sprintf("%s@%s", lp.Name, b.Currency)

	return &bus.B_LP_V2_GetPositionStatus_Response{
		Owner:             pos.Owner,
		ChainId:           pos.ChainId,
		Token0:            token0,
		Token1:            token1,
		Factory:           pos.Factory,
		Pair:              pos.Pair,
		LPBalance:         lpBalance,
		TotalSupply:       totalSupply,
		Reserve0:          reserve0,
		Reserve1:          reserve1,
		Liquidity0:        liquidity0,
		Liquidity1:        liquidity1,
		Liquidity0Dollars: liquidity0Dollars,
		Liquidity1Dollars: liquidity1Dollars,
		ProviderName:      pn,
	}, nil
}

func getToken0(chainId int, pair common.Address) (common.Address, error) {
	data, err := V2_PAIR.Pack("token0")
	if err != nil {
		return common.Address{}, err
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: chainId,
		To:      pair,
		Data:    data,
	})

	if resp.Error != nil {
		return common.Address{}, resp.Error
	}

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		return common.Address{}, err
	}

	var token0 common.Address
	err = V2_PAIR.UnpackIntoInterface(&token0, "token0", output)
	if err != nil {
		return common.Address{}, err
	}

	return token0, nil
}

func getToken1(chainId int, pair common.Address) (common.Address, error) {
	data, err := V2_PAIR.Pack("token1")
	if err != nil {
		return common.Address{}, err
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: chainId,
		To:      pair,
		Data:    data,
	})

	if resp.Error != nil {
		return common.Address{}, resp.Error
	}

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		return common.Address{}, err
	}

	var token1 common.Address
	err = V2_PAIR.UnpackIntoInterface(&token1, "token1", output)
	if err != nil {
		return common.Address{}, err
	}

	return token1, nil
}

func getBalanceOf(chainId int, pair common.Address, owner common.Address) (*big.Int, error) {
	data, err := V2_PAIR.Pack("balanceOf", owner)
	if err != nil {
		return nil, err
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: chainId,
		To:      pair,
		Data:    data,
	})

	if resp.Error != nil {
		return nil, resp.Error
	}

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		return nil, err
	}

	var balance *big.Int
	err = V2_PAIR.UnpackIntoInterface(&balance, "balanceOf", output)
	if err != nil {
		return nil, err
	}

	return balance, nil
}

func getTotalSupply(chainId int, pair common.Address) (*big.Int, error) {
	data, err := V2_PAIR.Pack("totalSupply")
	if err != nil {
		return nil, err
	}

	resp := bus.Fetch("eth", "call", &bus.B_EthCall{
		ChainId: chainId,
		To:      pair,
		Data:    data,
	})

	if resp.Error != nil {
		return nil, resp.Error
	}

	output, err := hexutil.Decode(resp.Data.(string))
	if err != nil {
		return nil, err
	}

	var totalSupply *big.Int
	err = V2_PAIR.UnpackIntoInterface(&totalSupply, "totalSupply", output)
	if err != nil {
		return nil, err
	}

	return totalSupply, nil
}
