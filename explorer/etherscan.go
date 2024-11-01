package explorer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

type EtherScanAPI struct {
}

func (e *EtherScanAPI) DownloadContract(w *cmn.Wallet, b *cmn.Blockchain, a common.Address) (string, error) {
	if b.ExplorerAPIUrl == "" {
		return "", errors.New("blockchain has no explorer API")
	}

	err := DownloadContractABI(b, a)
	if err != nil {
		log.Error().Err(err).Msg("Error downloading contract ABI")
		return "", err
	}

	name, err := DownloadContractCode(b, a)
	if err != nil {
		log.Error().Err(err).Msg("Error downloading contract code")
		return "", err
	}

	return name, nil
}

func DownloadContractABI(b *cmn.Blockchain, a common.Address) error {
	if b.ExplorerUrl == "" {
		return errors.New("blockchain has no explorer")
	}

	exu, _ := strings.CutSuffix(b.ExplorerAPIUrl, "/")
	URL := fmt.Sprintf("%s?module=contract&action=getabi&address=%s&apikey=%s", exu, a.Hex(), b.ExplorerAPIToken)

	log.Trace().Msgf("Downloading ABI from %s", URL)

	resp, err := http.Get(URL)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting ABI from %s", b.ExplorerAPIUrl)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msgf("Error reading ABI from %s", b.ExplorerAPIUrl)
		return err
	}

	// Parse the JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Error().Err(err).Msg("Error parsing JSON response")
		return err
	}

	if result["status"] != "1" {
		log.Error().Msgf("API error: %s", result["message"])
		return fmt.Errorf("API error: %s", result["message"])
	}

	abi, ok := result["result"].(string)
	if !ok {
		log.Error().Msg("ABI not found in response")
		return fmt.Errorf("ABI not found in response")
	}

	contractDir := cmn.DataFolder + "/contracts/" + a.Hex()
	err = os.MkdirAll(contractDir, 0755) // Create the /contract/address directory if it doesn't exist
	if err != nil {
		return err
	}

	path := contractDir + "/abi.json"

	err = os.WriteFile(path, []byte(abi), 0644)
	if err != nil {
		log.Error().Err(err).Msgf("Error saving ABI to %s", path)
		return err
	}

	bus.Send("ui", "notify", "ABI saved")

	return nil
}

func DownloadContractCode(b *cmn.Blockchain, a common.Address) (string, error) {

	exu, _ := strings.CutSuffix(b.ExplorerAPIUrl, "/")
	URL := fmt.Sprintf("%s?module=contract&action=getsourcecode&address=%s&apikey=%s", exu, a.Hex(), b.ExplorerAPIToken)

	log.Trace().Msgf("Downloading contract from %s", URL)

	resp, err := http.Get(URL)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting contract code from %s", b.ExplorerAPIUrl)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msgf("Error reading response from %s", b.ExplorerAPIUrl)
		return "", err
	}

	// Parse the JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Error().Err(err).Msg("Error parsing JSON response")
		return "", err
	}

	if result["status"] != "1" {
		log.Error().Msgf("API error: %s", result["message"])
		return "", fmt.Errorf("API error: %s", result["message"])
	}

	// The result is a slice with one element
	resultArray, ok := result["result"].([]interface{})
	if !ok || len(resultArray) == 0 {
		log.Error().Msg("Unexpected result format")
		return "", fmt.Errorf("unexpected result format")
	}

	contractInfo, ok := resultArray[0].(map[string]interface{})
	if !ok {
		log.Error().Msg("Unexpected contract info format")
		return "", fmt.Errorf("unexpected contract info format")
	}

	name, ok := contractInfo["ContractName"].(string)
	if !ok {
		log.Error().Msg("ContractName not found in response")
		return "", fmt.Errorf("contract name not found in response")
	}

	sourceCode, ok := contractInfo["SourceCode"].(string)
	if !ok {
		log.Error().Msg("SourceCode not found in response")
		return "", fmt.Errorf("source code not found in response")
	}

	// Create directories
	contractSrcDir := cmn.DataFolder + "/contracts/" + a.Hex() + "/src"
	err = os.MkdirAll(contractSrcDir, 0755) // Create the /contract/address directory if it doesn't exist
	if err != nil {
		return "", err
	}

	// Handle multi-file contracts (SourceCode is a JSON string)
	if len(sourceCode) > 0 && sourceCode[0] == '{' {
		// Parse the JSON string
		var sourceCodeJSON map[string]interface{}

		if strings.HasPrefix(sourceCode, "{{") {
			sourceCode = strings.TrimPrefix(sourceCode, "{")
			sourceCode = strings.TrimSuffix(sourceCode, "}")
		}

		if err := json.Unmarshal([]byte(sourceCode), &sourceCodeJSON); err != nil {
			log.Error().Err(err).Msg("Error parsing SourceCode JSON")
			return "", err
		}

		sources, ok := sourceCodeJSON["sources"].(map[string]interface{})
		if !ok {
			log.Error().Msg("Sources not found in SourceCode JSON")
			return "", fmt.Errorf("sources not found in SourceCode JSON")
		}

		for fileName, fileInfo := range sources {
			contentInfo, ok := fileInfo.(map[string]interface{})
			if !ok {
				log.Error().Msgf("Invalid content format for file: %s", fileName)
				continue
			}
			content, ok := contentInfo["content"].(string)
			if !ok {
				log.Error().Msgf("Content not found for file: %s", fileName)
				continue
			}

			filePath := fmt.Sprintf("%s/%s", contractSrcDir, fileName)

			fdir := filepath.Dir(filePath)
			err = os.MkdirAll(fdir, 0755)
			if err != nil {
				log.Error().Err(err).Msgf("Error creating directories : %s", fdir)
				return "", err
			}

			err = os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				log.Error().Err(err).Msgf("Error saving source code to %s", filePath)
				return "", err
			}
		}
	} else {
		filePath := fmt.Sprintf("%s/contract.sol", contractSrcDir)

		err = os.WriteFile(filePath, []byte(sourceCode), 0644)
		if err != nil {
			log.Error().Err(err).Msgf("Error saving contract code to %s", filePath)
			return "", err
		}

	}

	bus.Send("ui", "notify", "Contract code saved")

	return name, nil
}
