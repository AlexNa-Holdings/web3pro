package command

import (
	"strconv"
	"strings"
	"time"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
)

var config_subcommands = []string{"set"}

var config_params = []string{
	"verbosity",
	"theme",
	"bus_timeout",
	"bus_hard_timeout",
	"price_update_period",
	"editor",
	"cmc_api_key",
	"thegraph_api_key",
	"thegraph_gateway",
	"min_token_value",
}

func NewConfigCommand() *Command {
	return &Command{
		Command:      "config",
		ShortCommand: "cfg",
		Subcommands:  config_subcommands,
		Usage: `
Usage: config [COMMAND]

Manage application configuration.

COMMANDS:
  (no command)          - show current configuration
  set PARAM VALUE       - set configuration parameter

PARAMETERS:
  verbosity             - log level (trace/debug/info/warn/error/fatal/panic)
  theme                 - UI theme name
  bus_timeout           - bus request timeout (e.g., 3m, 180s)
  bus_hard_timeout      - bus hard timeout (e.g., 5m, 300s)
  price_update_period   - price update interval (e.g., 15m, 1h)
  editor                - external editor command
  cmc_api_key           - CoinMarketCap API key
  thegraph_api_key      - The Graph API key
  thegraph_gateway      - The Graph gateway URL
  min_token_value       - minimum USD value to show token in tokens pane
`,
		Help:             `Application configuration management`,
		Process:          Config_Process,
		AutoCompleteFunc: Config_AutoComplete,
	}
}

func Config_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}
	p := cmn.Split3(input)
	command, subcommand, param := p[0], p[1], p[2]

	if subcommand == "" || (subcommand != "set" && !strings.HasPrefix("set", subcommand)) {
		options = append(options, ui.ACOption{Name: "set", Result: command + " set "})
		return "action", &options, subcommand
	}

	if subcommand == "set" {
		// Autocomplete parameter name
		for _, p := range config_params {
			if param == "" || strings.Contains(p, param) {
				options = append(options, ui.ACOption{Name: p, Result: command + " set " + p + " "})
			}
		}
		return "parameter", &options, param
	}

	return "", &options, ""
}

func Config_Process(cmd *Command, input string) {
	tokens := strings.Fields(input)

	if len(tokens) < 2 {
		// Show current config
		showConfig()
		return
	}

	subcommand := tokens[1]

	switch subcommand {
	case "set":
		if len(tokens) < 4 {
			ui.PrintErrorf("Usage: config set PARAM VALUE\n")
			return
		}
		param := tokens[2]
		value := strings.Join(tokens[3:], " ")
		setConfigParam(param, value)
	default:
		ui.PrintErrorf("Unknown subcommand: %s\n", subcommand)
	}
}

func showConfig() {
	ui.Printf("\nCurrent configuration:\n\n")
	ui.Printf("  %-20s %s\n", "verbosity:", cmn.Config.Verbosity)
	ui.Printf("  %-20s %s\n", "theme:", cmn.Config.Theme)
	ui.Printf("  %-20s %s\n", "bus_timeout:", cmn.Config.BusTimeout.String())
	ui.Printf("  %-20s %s\n", "bus_hard_timeout:", cmn.Config.BusHardTimeout.String())
	ui.Printf("  %-20s %s\n", "price_update_period:", cmn.Config.PriceUpdatePeriod)
	ui.Printf("  %-20s %s\n", "editor:", cmn.Config.Editor)

	// Mask API keys
	cmcKey := cmn.Config.CMC_API_KEY
	if len(cmcKey) > 8 {
		cmcKey = cmcKey[:4] + "..." + cmcKey[len(cmcKey)-4:]
	} else if cmcKey != "" {
		cmcKey = "***"
	}
	ui.Printf("  %-20s %s\n", "cmc_api_key:", cmcKey)

	graphKey := cmn.Config.TheGraphAPIKey
	if len(graphKey) > 8 {
		graphKey = graphKey[:4] + "..." + graphKey[len(graphKey)-4:]
	} else if graphKey != "" {
		graphKey = "***"
	}
	ui.Printf("  %-20s %s\n", "thegraph_api_key:", graphKey)

	ui.Printf("  %-20s %s\n", "thegraph_gateway:", cmn.Config.TheGraphGateway)
	ui.Printf("  %-20s %.2f\n", "min_token_value:", cmn.Config.MinTokenValue)
	ui.Printf("\n")
}

func setConfigParam(param, value string) {
	var err error

	switch param {
	case "verbosity":
		validLevels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
		if !cmn.IsInArray(validLevels, value) {
			ui.PrintErrorf("Invalid verbosity level. Valid: %s\n", strings.Join(validLevels, ", "))
			return
		}
		cmn.Config.Verbosity = value

	case "theme":
		if _, ok := ui.Themes[value]; !ok {
			ui.PrintErrorf("Unknown theme: %s\n", value)
			return
		}
		cmn.Config.Theme = value
		ui.SetTheme(value)
		ui.Gui.DeleteAllViews()

	case "bus_timeout":
		duration, err := time.ParseDuration(value)
		if err != nil {
			ui.PrintErrorf("Invalid duration: %v\n", err)
			return
		}
		cmn.Config.BusTimeout = duration

	case "bus_hard_timeout":
		duration, err := time.ParseDuration(value)
		if err != nil {
			ui.PrintErrorf("Invalid duration: %v\n", err)
			return
		}
		cmn.Config.BusHardTimeout = duration

	case "price_update_period":
		// Validate by parsing
		_, err := time.ParseDuration(value)
		if err != nil {
			ui.PrintErrorf("Invalid duration: %v\n", err)
			return
		}
		cmn.Config.PriceUpdatePeriod = value

	case "editor":
		cmn.Config.Editor = value

	case "cmc_api_key":
		cmn.Config.CMC_API_KEY = value

	case "thegraph_api_key":
		cmn.Config.TheGraphAPIKey = value

	case "thegraph_gateway":
		cmn.Config.TheGraphGateway = value

	case "min_token_value":
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			ui.PrintErrorf("Invalid number: %v\n", err)
			return
		}
		if val < 0 {
			ui.PrintErrorf("Value must be >= 0\n")
			return
		}
		cmn.Config.MinTokenValue = val

	default:
		ui.PrintErrorf("Unknown parameter: %s\n", param)
		return
	}

	// Save config
	cmn.ConfigChanged = true
	err = cmn.SaveConfig()
	if err != nil {
		ui.PrintErrorf("Failed to save config: %v\n", err)
		return
	}

	ui.Printf("Config updated: %s = %s\n", param, value)

	// Reload config to apply changes
	err = cmn.RestoreConfig(cmn.ConfPath)
	if err != nil {
		ui.PrintErrorf("Failed to reload config: %v\n", err)
	}
}
