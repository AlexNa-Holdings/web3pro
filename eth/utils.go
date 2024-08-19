package eth

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
)

func BalanceOf(b *cmn.Blockchain, t *cmn.Token, address common.Address) (*big.Int, error) {
	if b.Name != t.Blockchain {
		return nil, fmt.Errorf("BalanceOf: Token (%s) is not from blockchain (%s)", t.Name, b.Name)
	}

	if t.Native {
		return GetBalance(b, address)
	} else {
		return GetERC20Balance(b, t, address)
	}
}
