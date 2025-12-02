package eth

import (
	"context"
	"errors"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
)

func GetERC20TokenInfo(b *cmn.Blockchain, address common.Address) (string, string, int, error) {

	client, err := getEthClient(b)
	if err != nil {
		log.Error().Msgf("GetERC20TokenInfo: Failed to open client: %v", err)
		return "", "", 0, err
	}

	msg := ethereum.CallMsg{
		To: &address,
	}

	// Get name
	msg.Data, err = ERC20_ABI.Pack("name")
	if err != nil {
		log.Error().Msgf("GetERC20TokenInfo: Cannot pack name. Error:(%v)", err)
		return "", "", 0, err
	}
	output, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Error().Msgf("GetERC20TokenInfo: Cannot call name. Error:(%v)", err)
		return "", "", 0, err
	}
	var decodedResult struct {
		Result string
	}
	err = ERC20_ABI.UnpackIntoInterface(&decodedResult, "name", output)
	if err != nil {
		log.Error().Msgf("GetERC20TokenInfo: Cannot unpack name. Error:(%v)", err)
		return "", "", 0, err
	}
	name := decodedResult.Result

	// Get symbol
	msg.Data, err = ERC20_ABI.Pack("symbol")
	if err != nil {
		log.Error().Msgf("GetERC20TokenInfo: Cannot pack symbol. Error:(%v)", err)
		return "", "", 0, err
	}
	output, err = client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Error().Msgf("GetERC20TokenInfo: Cannot call symbol. Error:(%v)", err)
		return "", "", 0, err
	}
	err = ERC20_ABI.UnpackIntoInterface(&decodedResult, "symbol", output)
	if err != nil {
		log.Error().Msgf("GetERC20TokenInfo: Cannot unpack symbol. Error:(%v)", err)
		return "", "", 0, err
	}
	symbol := decodedResult.Result

	// Get decimals
	msg.Data, err = ERC20_ABI.Pack("decimals")
	if err != nil {
		log.Error().Msgf("GetERC20TokenInfo: Cannot pack decimals. Error:(%v)", err)
		return "", "", 0, err
	}
	output, err = client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Error().Msgf("GetERC20TokenInfo: Cannot call decimals. Error:(%v)", err)
		return "", "", 0, err
	}
	var decodedResultD struct {
		Result uint8
	}
	err = ERC20_ABI.UnpackIntoInterface(&decodedResultD, "decimals", output)
	if err != nil {
		log.Error().Msgf("GetERC20TokenInfo: Cannot unpack decimals. Error:(%v)", err)
		return "", "", 0, err
	}
	decimals := int(decodedResultD.Result)

	return symbol, name, decimals, nil
}

func GetERC20Balance(b *cmn.Blockchain, t *cmn.Token, address common.Address) (*big.Int, error) {

	if t.ChainId != b.ChainId {
		log.Error().Msgf("GetERC20Balance: Token blockchain mismatch. Token:(%d) Blockchain:(%d)", t.ChainId, b.ChainId)
		return nil, nil
	}

	client, err := getEthClient(b)
	if err != nil {
		log.Error().Msgf("GetERC20Balance: Failed to open client: %v", err)
		return nil, err
	}

	msg := ethereum.CallMsg{
		To: &t.Address,
	}

	msg.Data, err = ERC20_ABI.Pack("balanceOf", address)
	if err != nil {
		log.Error().Msgf("GetERC20Balance: Cannot pack data. Error:(%v)", err)
		return nil, err
	}

	output, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Error().Msgf("GetERC20Balance: Cannot call contract. Error:(%v)", err)
		return nil, err
	}

	var decodedResult struct {
		Balance *big.Int
	}

	err = ERC20_ABI.UnpackIntoInterface(&decodedResult, "balanceOf", output)
	if err != nil {
		log.Error().Msgf("GetERC20Balance: Cannot unpack data. Error:(%v)", err)
		return nil, err
	}

	return decodedResult.Balance, nil
}

func BuildTxERC20Transfer(b *cmn.Blockchain, t *cmn.Token, s *cmn.Signer, from *cmn.Address,
	to common.Address, amount *big.Int) (*types.Transaction, error) {

	if t.ChainId != b.ChainId {
		log.Error().Msgf("BuildTxERC20Transfer: Token blockchain mismatch. Token:(%d) Blockchain:(%d)", t.ChainId, b.ChainId)
		return nil, errors.New("token blockchain mismatch")
	}

	if from.Signer != s.Name {
		log.Error().Msgf("BuildTxERC20Transfer: Signer mismatch. Token:(%s) Blockchain:(%s)", from.Signer, s.Name)
		return nil, errors.New("signer mismatch")
	}

	data, err := ERC20_ABI.Pack("transfer", to, amount)
	if err != nil {
		log.Error().Msgf("BuildTxERC20Transfer: Cannot pack data. Error:(%v)", err)
		return nil, err
	}

	client, err := getEthClient(b)
	if err != nil {
		log.Error().Msgf("BuildTxTransfer: Failed to open client: %v", err)
		return nil, err
	}

	nonce, err := client.PendingNonceAt(context.Background(), from.Address)
	if err != nil {
		log.Error().Msgf("BuildTxTransfer: Cannot get nonce. Error:(%v)", err)
		return nil, err
	}

	gasLimit, err := client.EstimateGas(context.Background(), ethereum.CallMsg{
		From: from.Address,
		To:   &t.Address,
		Data: data,
		Gas:  0,
	})
	if err != nil {
		log.Error().Msgf("BuildTxERC20Transfer: Cannot estimate gas. Error:(%v)", err)
		return nil, err
	}

	log.Debug().Msgf("BuildTxERC20Transfer: Gas limit: %v", gasLimit)

	// // Suggest gas price
	// gasPrice, err := client.SuggestGasPrice(context.Background())
	// if err != nil {
	// 	log.Error().Msgf("BuildTxTransfer: Cannot suggest gas price. Error:(%v)", err)
	// 	return nil, err
	// }

	priorityFee, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("Failed to suggest gas tip cap")
		return nil, err
	}

	// Get the latest block to determine the base fee
	block, err := client.BlockByNumber(context.Background(), nil) // Get the latest block
	if err != nil {
		log.Error().Err(err).Msg("Failed to get the latest block")
		return nil, err
	}

	// Base fee is included in the block header (introduced in EIP-1559)
	baseFee := block.BaseFee()
	// Calculate the MaxFeePerGas based on base fee and priority fee
	// For example, you might want to set MaxFeePerGas to be slightly higher than baseFee + priorityFee
	maxFeePerGas := new(big.Int).Add(baseFee, priorityFee)
	buffer := big.NewInt(2) // Set a buffer (optional) to ensure transaction gets processed
	maxFeePerGas = maxFeePerGas.Mul(maxFeePerGas, buffer)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(int64(b.ChainId)),
		Nonce:     nonce,
		To:        &t.Address,
		Value:     big.NewInt(0),
		Gas:       gasLimit,
		GasFeeCap: maxFeePerGas,
		GasTipCap: priorityFee,
		Data:      data,
	})

	return tx, nil

}

// BalanceQuery represents a single balance query (token + holder address)
type BalanceQuery struct {
	Token   *cmn.Token
	Holder  common.Address
	Balance *big.Int // Result will be stored here
}

// GetERC20BalancesBatch fetches multiple ERC20 balances in a single multicall
// All tokens must be on the same chain
func GetERC20BalancesBatch(b *cmn.Blockchain, queries []*BalanceQuery) error {
	if len(queries) == 0 {
		return nil
	}

	// Prepare multicall data
	calls := make([]bus.B_EthMultiCall_Call, len(queries))
	for i, q := range queries {
		if q.Token.ChainId != b.ChainId {
			log.Error().Msgf("GetERC20BalancesBatch: Token %s chainId %d != blockchain chainId %d",
				q.Token.Symbol, q.Token.ChainId, b.ChainId)
			continue
		}

		data, err := ERC20_ABI.Pack("balanceOf", q.Holder)
		if err != nil {
			log.Error().Err(err).Msgf("GetERC20BalancesBatch: Cannot pack balanceOf for %s", q.Token.Symbol)
			continue
		}

		calls[i] = bus.B_EthMultiCall_Call{
			To:   q.Token.Address,
			Data: data,
		}
	}

	// Execute multicall
	resp := bus.Fetch("eth", "multi-call", &bus.B_EthMultiCall{
		ChainId: b.ChainId,
		Calls:   calls,
	})

	if resp.Error != nil {
		log.Debug().Err(resp.Error).Msgf("GetERC20BalancesBatch: multicall failed for chain %d, falling back to individual calls", b.ChainId)
		// Fallback to individual calls
		for _, q := range queries {
			if q.Token.Native {
				balance, err := GetBalance(b, q.Holder)
				if err != nil {
					log.Debug().Err(err).Msgf("GetERC20BalancesBatch: fallback GetBalance failed for %s", q.Token.Symbol)
					q.Balance = big.NewInt(0)
				} else {
					q.Balance = balance
				}
			} else {
				balance, err := GetERC20Balance(b, q.Token, q.Holder)
				if err != nil {
					log.Debug().Err(err).Msgf("GetERC20BalancesBatch: fallback GetERC20Balance failed for %s", q.Token.Symbol)
					q.Balance = big.NewInt(0)
				} else {
					q.Balance = balance
				}
			}
		}
		return nil
	}

	results, ok := resp.Data.([][]byte)
	if !ok {
		log.Error().Msg("GetERC20BalancesBatch: Cannot convert multicall result")
		return errors.New("cannot convert multicall result")
	}

	if len(results) != len(queries) {
		log.Error().Msgf("GetERC20BalancesBatch: result count mismatch: got %d, expected %d", len(results), len(queries))
		return errors.New("multicall result count mismatch")
	}

	// Unpack results
	for i, result := range results {
		if len(result) == 0 {
			queries[i].Balance = big.NewInt(0)
			continue
		}

		var decodedResult struct {
			Balance *big.Int
		}
		err := ERC20_ABI.UnpackIntoInterface(&decodedResult, "balanceOf", result)
		if err != nil {
			log.Debug().Err(err).Msgf("GetERC20BalancesBatch: Cannot unpack balance for %s", queries[i].Token.Symbol)
			queries[i].Balance = big.NewInt(0)
			continue
		}
		queries[i].Balance = decodedResult.Balance
	}

	return nil
}

func ERC20Transfer(msg *bus.Message, b *cmn.Blockchain, t *cmn.Token, s *cmn.Signer, from *cmn.Address, to common.Address, amount *big.Int) error {
	log.Trace().Msgf("ERC20Transfer: Token:(%s) Blockchain:(%s) From:(%s) To:(%s) Amount:(%s)", t.Name, b.Name, from.Address.String(), to.String(), amount.String())

	tx, err := BuildTxERC20Transfer(b, t, s, from, to, amount)
	if err != nil {
		log.Error().Msgf("ERC20Transfer: Cannot build transaction. Error:(%v)", err)
		return err
	}

	res := msg.Fetch("signer", "sign-tx", &bus.B_SignerSignTx{
		Type:      s.Type,
		Name:      s.Name,
		MasterKey: s.MasterKey,
		Chain:     b.Name,
		Tx:        tx,
		From:      from.Address,
		Path:      from.Path,
	})

	if res.Error != nil {
		log.Error().Err(res.Error).Msg("ERC20Transfer: Cannot sign tx")
		return res.Error
	}

	signedTx, ok := res.Data.(*types.Transaction)
	if !ok {
		log.Error().Msgf("ERC20Transfer: Cannot convert to sig. Data:(%v)", res.Data)
		return errors.New("cannot convert to transaction")
	}

	hash, err := SendSignedTx(signedTx)
	if err != nil {
		log.Error().Err(err).Msg("ERC20Transfer: Cannot send tx")
		return err
	}

	bus.Send("ui", "notify", "Transaction sent: "+hash)

	return nil
}
