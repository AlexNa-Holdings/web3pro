package cmn

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

const VERSION = "0.1.0"
const LOG_NAME = "web3pro.log"
const CONFIG_NAME = "config.yaml"

var DataFolder = "data"
var AppName = "web3pro"
var LogPath = LOG_NAME
var ConfPath = CONFIG_NAME

var ConfigChanged = false

type SConfig struct {
	Verbosity            string        `yaml:"verbosity"`              // log verbosity
	Theme                string        `yaml:"theme"`                  // UI theme
	BusTimeout           time.Duration `yaml:"bus_timeout"`            // timeout for bus requests
	BusHardTimeout       time.Duration `yaml:"bus_hard_timeout"`       // hard timeout for bus requests
	PriceUpdatePeriod    string        `yaml:"price_update_period"`    // price update period
	Editor               string        `yaml:"editor"`                 // editor
	CMC_API_KEY          string        `yaml:"cmc_api_key"`            // CoinMarketCap API key
	TheGraphAPIKey       string        `yaml:"thegraph_api_key"`       // The Graph API key for subgraph queries
	TheGraphGateway      string        `yaml:"thegraph_gateway"`       // The Graph gateway URL (default: https://gateway.thegraph.com/api/{api-key}/subgraphs/id/)
	MinTokenValue        float64       `yaml:"min_token_value"`        // minimum USD value to show token in tokens pane
	WSEnabled            bool          `yaml:"ws_enabled"`             // enable WebSocket server for browser communication
}

var Config *SConfig = &SConfig{ //Default config
	Verbosity:         "debug",
	Theme:             "dark",
	BusTimeout:        3 * time.Minute,
	BusHardTimeout:    5 * time.Minute,
	PriceUpdatePeriod: "15m",
	Editor:            "code",
	TheGraphGateway:   "https://gateway.thegraph.com/api/{api-key}/subgraphs/id/",
	MinTokenValue:     1,
	WSEnabled:         true,
}

func InitConfig() {
	var err error

	// Get the data folder
	DataFolder, err = GetDataFolder()
	if err != nil {
		fmt.Printf("error getting data folder: %v", err)
		os.Exit(1)
	}

	// Init logger
	LogPath = filepath.Join(DataFolder, LOG_NAME)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// logFile, err := os.OpenFile(LogPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	logFile, err := os.OpenFile(LogPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666) // truncate log file
	if err != nil {
		log.Fatal().Msgf("error opening log file: %v", err)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: logFile})

	//Restore config from yaml file
	ConfPath = filepath.Join(DataFolder, CONFIG_NAME)
	err = RestoreConfig(ConfPath)
	if err != nil {
		log.Error().Msgf("error restoring config: %v", err)
	}

	switch Config.Verbosity {
	case "trace":
		log.Level(zerolog.TraceLevel)
	case "debug":
		log.Level(zerolog.DebugLevel)
	case "info":
		log.Level(zerolog.InfoLevel)
	case "warn":
		log.Level(zerolog.WarnLevel)
	case "error":
		log.Level(zerolog.ErrorLevel)
	case "fatal":
		log.Level(zerolog.FatalLevel)
	case "panic":
		log.Level(zerolog.PanicLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Info().Msgf("Log level: %s", Config.Verbosity)

	//create wallets folder if needed
	err = os.MkdirAll(filepath.Join(DataFolder, "wallets"), os.ModePerm)
	if err != nil {
		log.Error().Msgf("error creating wallet folder: %v", err)
	}

	log.Trace().Msg("Started")

}

func SaveConfig() error {
	if !ConfigChanged {
		return nil
	}

	data, err := yaml.Marshal(Config)
	if err != nil {
		return err
	}

	err = os.WriteFile(ConfPath, data, 0666)
	if err != nil {
		return err
	}

	ConfigChanged = false
	return err
}

func RestoreConfig(path string) error {
	data, err := os.ReadFile(DataFolder + "/config.yaml")
	if err != nil {
		if err != os.ErrNotExist {
			// it is ok. Let's use default config
			log.Warn().Msgf("no config file found: %v", err)
			return nil
		} else {
			return err
		}
	}

	err = yaml.Unmarshal(data, Config)
	if err != nil {
		return err
	}

	return err
}

func GetDataFolder() (string, error) {
	var dataDir string

	switch runtime.GOOS {
	case "windows":
		// Get the local app data folder
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return "", fmt.Errorf("LOCALAPPDATA environment variable is not set")
		}
		dataDir = filepath.Join(localAppData, AppName)
	case "darwin":
		// Get the user's home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("error getting home directory: %v", err)
		}
		dataDir = filepath.Join(homeDir, "Library", "Application Support", AppName)
	case "linux":
		// Get the user's home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("error getting home directory: %v", err)
		}
		dataDir = filepath.Join(homeDir, "."+AppName)
	default:
		return "", fmt.Errorf("unsupported operating system")
	}

	// Create the directory if it doesn't exist
	err := os.MkdirAll(dataDir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("error creating data directory: %v", err)
	}

	return dataDir, nil
}
