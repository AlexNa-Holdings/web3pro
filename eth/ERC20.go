package eth

import (
	"context"
	"fmt"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
)

func OpenClient(b *cmn.Blockchain) error {

	if b.Client != nil {
		return nil
	}

	client, err := ethclient.Dial(b.Url)
	if err != nil {
		return fmt.Errorf("OpenClient: Cannot dial to (%s). Error:(%v)", b.Url, err)
	}

	b.Client = client
	return nil
}

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
