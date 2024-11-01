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

	msg.Data, err = ERC20_ABI.Pack("name")
	if err != nil {
		log.Error().Msgf("ConfigGetAddr: Cannot pack data. Error:(%v)", err)
	}
	output, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Error().Msgf("ConfigGetAddr: Cannot call contract. Error:(%v)", err)
	}
	var decodedResult struct {
		Result string
	}
	err = ERC20_ABI.UnpackIntoInterface(&decodedResult, "name", output)
	if err != nil {
		log.Error().Msgf("ConfigGetAddr:Cannot unpack data. Error:(%v)", err)
	}
	name := decodedResult.Result

	msg.Data, err = ERC20_ABI.Pack("symbol")
	if err != nil {
		log.Error().Msgf("ConfigGetAddr: Cannot pack data. Error:(%v)", err)
	}
	output, err = client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Error().Msgf("ConfigGetAddr: Cannot call contract. Error:(%v)", err)
	}
	err = ERC20_ABI.UnpackIntoInterface(&decodedResult, "symbol", output)
	if err != nil {
		log.Error().Msgf("ConfigGetAddr:Cannot unpack data. Error:(%v)", err)
	}
	symbol := decodedResult.Result

	msg.Data, err = ERC20_ABI.Pack("decimals")
	if err != nil {
		log.Error().Msgf("ConfigGetAddr: Cannot pack data. Error:(%v)", err)
	}
	output, err = client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Error().Msgf("ConfigGetAddr: Cannot call contract. Error:(%v)", err)
	}
	var decodedResultD struct {
		Result uint8
	}
	err = ERC20_ABI.UnpackIntoInterface(&decodedResultD, "decimals", output)
	if err != nil {
		log.Error().Msgf("ConfigGetAddr:Cannot unpack data. Error:(%v)", err)
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
