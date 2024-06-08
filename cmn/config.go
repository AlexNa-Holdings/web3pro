package cmn

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

const VERSION = "0.0.1"
const LOG_NAME = "web3pro.log"
const CONFIG_NAME = "config.yaml"

var DataFolder = "data"
var AppName = "web3pro"
var LogPath = LOG_NAME
var ConfPath = CONFIG_NAME

var ConfigChanged = false

type SConfig struct {
	WalletName string `yaml:"wallet_name"` // last wallet used
	Verbosity  string `yaml:"verbosity"`   // log verbosity
}

var Config *SConfig = &SConfig{
	WalletName: "default",
	Verbosity:  "trace",
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
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		if err != os.ErrNotExist {
			// it is ok. Let's use default config
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