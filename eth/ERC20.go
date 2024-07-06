package eth

import (
	"context"
	"math/big"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

func GetERC20TokenInfo(b *cmn.Blockchain, address *common.Address) (string, string, int, error) {

	err := OpenClient(b)
	if err != nil {
		log.Error().Msgf("GetERC20TokenInfo: Failed to open client: %v", err)
		return "", "", 0, err
	}

	msg := ethereum.CallMsg{
		To: address,
	}

	msg.Data, err = ERC20.Pack("name")
	if err != nil {
		log.Error().Msgf("ConfigGetAddr: Cannot pack data. Error:(%v)", err)
	}
	output, err := b.Client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Error().Msgf("ConfigGetAddr: Cannot call contract. Error:(%v)", err)
	}
	var decodedResult struct {
		Result string
	}
	err = ERC20.UnpackIntoInterface(&decodedResult, "name", output)
	if err != nil {
		log.Error().Msgf("ConfigGetAddr:Cannot unpack data. Error:(%v)", err)
	}
	name := decodedResult.Result

	msg.Data, err = ERC20.Pack("symbol")
	if err != nil {
		log.Error().Msgf("ConfigGetAddr: Cannot pack data. Error:(%v)", err)
	}
	output, err = b.Client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Error().Msgf("ConfigGetAddr: Cannot call contract. Error:(%v)", err)
	}
	err = ERC20.UnpackIntoInterface(&decodedResult, "symbol", output)
	if err != nil {
		log.Error().Msgf("ConfigGetAddr:Cannot unpack data. Error:(%v)", err)
	}
	symbol := decodedResult.Result

	msg.Data, err = ERC20.Pack("decimals")
	if err != nil {
		log.Error().Msgf("ConfigGetAddr: Cannot pack data. Error:(%v)", err)
	}
	output, err = b.Client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Error().Msgf("ConfigGetAddr: Cannot call contract. Error:(%v)", err)
	}
	var decodedResultD struct {
		Result uint8
	}
	err = ERC20.UnpackIntoInterface(&decodedResultD, "decimals", output)
	if err != nil {
		log.Error().Msgf("ConfigGetAddr:Cannot unpack data. Error:(%v)", err)
	}
	decimals := int(decodedResultD.Result)

	return symbol, name, decimals, nil
}

func GetERC20Balance(b *cmn.Blockchain, t *cmn.Token, address common.Address) (*big.Int, error) {

	if t.Blockchain != b.Name {
		log.Error().Msgf("GetERC20Balance: Token blockchain mismatch. Token:(%s) Blockchain:(%s)", t.Blockchain, b.Name)
		return nil, nil
	}

	err := OpenClient(b)
	if err != nil {
		log.Error().Msgf("GetERC20Balance: Failed to open client: %v", err)
		return nil, err
	}

	msg := ethereum.CallMsg{
		To: &t.Address,
	}

	msg.Data, err = ERC20.Pack("balanceOf", address)
	if err != nil {
		log.Error().Msgf("GetERC20Balance: Cannot pack data. Error:(%v)", err)
		return nil, err
	}

	output, err := b.Client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Error().Msgf("GetERC20Balance: Cannot call contract. Error:(%v)", err)
		return nil, err
	}

	var decodedResult struct {
		Balance *big.Int
	}

	err = ERC20.UnpackIntoInterface(&decodedResult, "balanceOf", output)
	if err != nil {
		log.Error().Msgf("GetERC20Balance: Cannot unpack data. Error:(%v)", err)
		return nil, err
	}

	return decodedResult.Balance, nil
}

func GetERC20Transfer(b *cmn.Blockchain, t *cmn.Token, s *cmn.Signer, from *cmn.Address, to common.Address, amount *big.Int) ([]byte, error) {

	if t.Blockchain != b.Name {
		log.Error().Msgf("GetERC20Balance: Token blockchain mismatch. Token:(%s) Blockchain:(%s)", t.Blockchain, b.Name)
		return nil, nil
	}

	if from.Signer != s.Name {
		log.Error().Msgf("GetERC20Transfer: Signer mismatch. Token:(%s) Blockchain:(%s)", from.Signer, s.Name)
		return nil, nil
	}

	err := OpenClient(b)
	if err != nil {
		log.Error().Msgf("GetERC20Transfer: Failed to open client: %v", err)
		return nil, err
	}

	msg := ethereum.CallMsg{
		From: from.Address,
		To:   &t.Address,
	}

	msg.Data, err = ERC20.Pack("transfer", to, amount)
	if err != nil {
		log.Error().Msgf("GetERC20Transfer: Cannot pack data. Error:(%v)", err)
		return nil, err
	}

	output, err := b.Client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Error().Msgf("GetERC20Transfer: Cannot call contract. Error:(%v)", err)
		return nil, err
	}

	var decodedResult struct {
		Result []byte
	}

	err = ERC20.UnpackIntoInterface(&decodedResult, "transfer", output)
	if err != nil {
		log.Error().Msgf("GetERC20Transfer: Cannot unpack data. Error:(%v)", err)
		return nil, err
	}

	return decodedResult.Result, nil
}
