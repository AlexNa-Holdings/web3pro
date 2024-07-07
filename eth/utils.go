package eth

import (
	"fmt"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
)

func OpenClient(b *cmn.Blockchain) error {

	log.Trace().Msgf("OpenClient: Opening client for (%s)", b.Name)

	if b.Client != nil {
		return nil
	}

	client, err := ethclient.Dial(b.Url)
	if err != nil {
		return fmt.Errorf("OpenClient: Cannot dial to (%s). Error:(%v)", b.Url, err)
	}

	log.Trace().Msgf("OpenClient: Client opened to (%s)", b.Url)

	b.Client = client
	return nil
}

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

// // Create a call message for estimating gas
// msg := ethereum.CallMsg{
//     From:     fromAddress,
//     To:       &toAddress,
//     Value:    value,
//     GasPrice: gasPrice,
//     Data:     nil,
// }

// // Estimate the gas required
// gasLimit, err := client.EstimateGas(context.Background(), msg)
// if err != nil {
//     log.Fatalf("Failed to estimate gas: %v", err)
// }
