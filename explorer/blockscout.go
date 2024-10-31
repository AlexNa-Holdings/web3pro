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

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

type SmartContract struct {
	VerifiedTwinAddressHash    string                 `json:"verified_twin_address_hash"`
	IsVerified                 bool                   `json:"is_verified"`
	IsChangedBytecode          bool                   `json:"is_changed_bytecode"`
	IsPartiallyVerified        bool                   `json:"is_partially_verified"`
	IsFullyVerified            bool                   `json:"is_fully_verified"`
	IsVerifiedViaSourcify      bool                   `json:"is_verified_via_sourcify"`
	IsVerifiedViaEthBytecodeDb bool                   `json:"is_verified_via_eth_bytecode_db"`
	IsVyperContract            bool                   `json:"is_vyper_contract"`
	IsSelfDestructed           bool                   `json:"is_self_destructed"`
	CanBeVisualizedViaSol2uml  bool                   `json:"can_be_visualized_via_sol2uml"`
	MinimalProxyAddressHash    string                 `json:"minimal_proxy_address_hash"`
	SourcifyRepoURL            string                 `json:"sourcify_repo_url"`
	Name                       string                 `json:"name"`
	OptimizationEnabled        bool                   `json:"optimization_enabled"`
	OptimizationsRuns          int                    `json:"optimizations_runs"`
	CompilerVersion            string                 `json:"compiler_version"`
	EvmVersion                 string                 `json:"evm_version"`
	VerifiedAt                 string                 `json:"verified_at"`
	Abi                        []ABIElement           `json:"abi"`
	SourceCode                 string                 `json:"source_code"`
	FilePath                   string                 `json:"file_path"`
	CompilerSettings           map[string]interface{} `json:"compiler_settings"`
	ConstructorArgs            string                 `json:"constructor_args"`
	AdditionalSources          []ContractSource       `json:"additional_sources"`
	// DecodedConstructorArgs     []ConstructorArguments `json:"decoded_constructor_args"`
	// DeployedBytecode           string                 `json:"deployed_bytecode"`
	// CreationBytecode           string                 `json:"creation_bytecode"`
	// ExternalLibraries          []ExternalLibrary      `json:"external_libraries"`
	// Language                   string                 `json:"language"`
}

type ABIElement struct {
	Type            string            `json:"type"`
	StateMutability string            `json:"stateMutability,omitempty"`
	Inputs          []ABIInputElement `json:"inputs,omitempty"`
	Name            string            `json:"name,omitempty"`
}

type ABIInputElement struct {
	Type         string            `json:"type"`
	Name         string            `json:"name"`
	InternalType string            `json:"internalType,omitempty"`
	Components   []ABIInputElement `json:"components,omitempty"`
}

type ContractSource struct {
	FilePath   string `json:"file_path"`
	SourceCode string `json:"source_code"`
}

type ConstructorArguments struct {
	// Define fields as needed
}

type ExternalLibrary struct {
	Name        string `json:"name"`
	AddressHash string `json:"address_hash"`
}

type BlockscoutAPI struct {
}

func (e *BlockscoutAPI) DownloadContract(w *cmn.Wallet, b *cmn.Blockchain, a common.Address) (string, error) {
	if b.ExplorerUrl == "" {
		return "", errors.New("blockchain has no explorer")
	}

	exu, _ := strings.CutSuffix(b.ExplorerAPIUrl, "/")

	URL := fmt.Sprintf("%s/smart-contracts/%s", exu, a.Hex())

	log.Trace().Msgf("Downloading contract from %s", URL)

	resp, err := http.Get(URL)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// log.Debug().Msgf("Downloaded contract: %s", string(body))

	var sc SmartContract
	err = json.Unmarshal(body, &sc)
	if err != nil {
		return "", err
	}

	contractDir := cmn.DataFolder + "/contracts/" + a.Hex()
	err = os.MkdirAll(contractDir, 0755) // Create the /contract/address directory if it doesn't exist
	if err != nil {
		return "", err
	}

	path := contractDir + "/abi.json"

	// Marshal the ABI slice to JSON
	abiData, err := json.MarshalIndent(sc.Abi, "", "  ")
	if err != nil {
		return "", err
	}

	err = os.WriteFile(path, abiData, 0644) // Save the ABI
	if err != nil {
		return "", err
	}

	contractSrcDir := contractDir + "/src"
	err = os.MkdirAll(contractSrcDir, 0755) // Create the /contract/address directory if it doesn't exist
	if err != nil {
		return "", err
	}

	for _, source := range sc.AdditionalSources {
		sourcePath := filepath.Join(contractSrcDir, source.FilePath)
		err = os.MkdirAll(filepath.Dir(sourcePath), 0755) // Ensure directories exist
		if err != nil {
			return "", err
		}

		err = os.WriteFile(sourcePath, []byte(source.SourceCode), 0644) // Save the source code
		if err != nil {
			return "", err
		}
	}

	// Save the main contract source code
	mainSourcePath := filepath.Join(contractSrcDir, sc.FilePath)

	err = os.MkdirAll(filepath.Dir(mainSourcePath), 0755)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(mainSourcePath, []byte(sc.SourceCode), 0644)
	if err != nil {
		return "", err
	}

	return sc.Name, nil
}
