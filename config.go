package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const VERSION = "0.0.1"
const LOG_NAME = "web3pro.log"

var DataFolder = "data"
var AppName = "web3pro"
var LogPath = LOG_NAME

type SConfig struct {
	WalletName string
	Verbosity  string
}

var Config *SConfig

func InitConfig() {
	var err error

	Config = &SConfig{}

	// Get the data folder
	DataFolder, err = GetDataFolder()
	if err != nil {
		log.Fatal().Msgf("error getting data folder: %v", err)
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
