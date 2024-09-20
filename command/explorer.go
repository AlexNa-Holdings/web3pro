package command

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

var explorer_subcommands = []string{"get_abi", "get_code"}

func NewExplorerCommand() *Command {
	return &Command{
		Command:      "explorer",
		ShortCommand: "x",
		Usage: `
Usage: explorer [command] [params]


Commands:

  abi [BLOCKCHAIN] [CONTRACT] - download ABI for contract
`,
		Help:             `Explorer API`,
		Process:          Explorer_Process,
		AutoCompleteFunc: Explorer_AutoComplete,
	}
}

func Explorer_AutoComplete(input string) (string, *[]ui.ACOption, string) {

	if cmn.CurrentWallet == nil {
		return "", nil, ""
	}

	w := cmn.CurrentWallet
	if w == nil {
		return "", nil, ""
	}

	options := []ui.ACOption{}
	p := cmn.SplitN(input, 6)
	command, subcommand, bchain, address := p[0], p[1], p[2], p[3]

	last_param := len(p) - 1
	for last_param > 0 && p[last_param] == "" {
		last_param--
	}

	if strings.HasSuffix(input, " ") {
		last_param++
	}

	b := w.GetBlockchain(bchain)

	b = b
	address = address

	switch last_param {
	case 0, 1:
		for _, sc := range explorer_subcommands {
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	case 2:
		for _, chain := range w.Blockchains {
			if cmn.Contains(chain.Name, bchain) {
				options = append(options, ui.ACOption{
					Name: chain.Name, Result: command + " " + subcommand + " '" + chain.Name + "' "})
			}
		}
		return "blockchain", &options, bchain
	}

	return "", nil, ""
}

func Explorer_Process(c *Command, input string) {

	if cmn.CurrentWallet == nil {
		ui.PrintErrorf("No wallet open")
		return
	}

	w := cmn.CurrentWallet
	if w == nil {
		ui.PrintErrorf("Explorer_Process: no wallet")
		return
	}

	p := cmn.SplitN(input, 6)
	_, subcommand, bchain, address := p[0], p[1], p[2], p[3]

	b := w.GetBlockchain(bchain)
	if b == nil {
		ui.PrintErrorf("Explorer_Process: blockchain not found: %v", bchain)
		return
	}

	if b.ExplorerUrl == "" {
		ui.PrintErrorf("Explorer_Process: blockchain %s has no explorer", b.Name)
		return
	}

	// check the address format
	if !common.IsHexAddress(address) {
		ui.PrintErrorf("Explorer_Process: invalid address: %s", address)
		return
	}

	switch subcommand {
	case "get_abi":
		downloadABI(b.ExplorerUrl, b.ExplorerAPIToken, address)
	case "get_code":
		downloadCode(b.ExplorerUrl, b.ExplorerAPIToken, address)
	}

}

func downloadABI(url string, token string, address string) error {
	apiURL, err := getAPIURL(url)
	if err != nil {
		log.Error().Err(err).Msg("Error getting API URL")
		return err
	}

	URL := fmt.Sprintf("%s?module=contract&action=getabi&address=%s&apikey=%s", apiURL, address, token)

	resp, err := http.Get(URL)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting ABI from %s", apiURL)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msgf("Error reading ABI from %s", apiURL)
		return err
	}

	os.MkdirAll(cmn.DataFolder+"/abi", 0755)

	path := cmn.DataFolder + "/abi/" + address + ".json"
	err = os.WriteFile(path, body, 0644)
	if err != nil {
		log.Error().Err(err).Msgf("Error saving ABI to %s", path)
		return err
	}

	ui.Printf("ABI saved to %s", path)

	return nil
}

func downloadCode(url string, token string, address string) error {
	apiURL, err := getAPIURL(url)
	if err != nil {
		log.Error().Err(err).Msg("Error getting API URL")
		return err
	}
	URL := fmt.Sprintf("%s?module=contract&action=getsourcecode&address=%s&apikey=%s", apiURL, address, token)

	resp, err := http.Get(URL)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting contract code from %s", apiURL)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msgf("Error reading response from %s", apiURL)
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

	// The result is a slice with one element
	resultArray, ok := result["result"].([]interface{})
	if !ok || len(resultArray) == 0 {
		log.Error().Msg("Unexpected result format")
		return fmt.Errorf("unexpected result format")
	}

	contractInfo, ok := resultArray[0].(map[string]interface{})
	if !ok {
		log.Error().Msg("Unexpected contract info format")
		return fmt.Errorf("unexpected contract info format")
	}

	sourceCode, ok := contractInfo["SourceCode"].(string)
	if !ok {
		log.Error().Msg("SourceCode not found in response")
		return fmt.Errorf("source code not found in response")
	}

	// Create directories
	dir := cmn.DataFolder + "/contracts/" + address
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		log.Error().Err(err).Msgf("Error creating directories : %s", dir)
		return err
	}

	// Handle multi-file contracts (SourceCode is a JSON string)
	if len(sourceCode) > 0 && sourceCode[0] == '{' {
		// Parse the JSON string
		var sourceCodeJSON map[string]interface{}
		if err := json.Unmarshal([]byte(sourceCode), &sourceCodeJSON); err != nil {
			log.Error().Err(err).Msg("Error parsing SourceCode JSON")
			return err
		}

		sources, ok := sourceCodeJSON["sources"].(map[string]interface{})
		if !ok {
			log.Error().Msg("Sources not found in SourceCode JSON")
			return fmt.Errorf("sources not found in SourceCode JSON")
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

			filePath := fmt.Sprintf("%s/%s", dir, fileName)
			err = os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				log.Error().Err(err).Msgf("Error saving source code to %s", filePath)
				return err
			}

			ui.Printf("Source code saved to %s", filePath)
		}
	} else {
		filePath := fmt.Sprintf("%s/contract.sol", dir)
		err = os.WriteFile(filePath, []byte(sourceCode), 0644)
		if err != nil {
			log.Error().Err(err).Msgf("Error saving contract code to %s", filePath)
			return err
		}

		ui.Printf("Contract code saved to %s", filePath)
	}

	return nil
}

func getAPIURL(explorerURL string) (string, error) {
	parsedURL, err := url.Parse(explorerURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %v", err)
	}

	// Extract host components
	host := parsedURL.Hostname()
	port := parsedURL.Port()

	// Split the host into parts
	hostParts := strings.Split(host, ".")
	if len(hostParts) < 2 {
		return "", fmt.Errorf("invalid host in URL")
	}

	// Remove 'www' if present
	if hostParts[0] == "www" {
		hostParts = hostParts[1:]
	}

	// Check if there's an existing subdomain (e.g., 'testnet')
	subdomain := ""
	if len(hostParts) > 2 {
		subdomain = hostParts[0]
		hostParts = hostParts[1:]
	}

	// Build the new host with 'api' subdomain
	newHostParts := []string{}
	if subdomain != "" {
		newHostParts = append(newHostParts, "api."+subdomain)
	} else {
		newHostParts = append(newHostParts, "api")
	}
	newHostParts = append(newHostParts, hostParts...)

	// Reconstruct the host
	newHost := strings.Join(newHostParts, ".")

	// Include port if present
	if port != "" {
		newHost = fmt.Sprintf("%s:%s", newHost, port)
	}

	// Set the new host in the URL
	parsedURL.Host = newHost

	// Ensure path starts with "/api"
	parsedURL.Path = "/api"

	return parsedURL.String(), nil
}
