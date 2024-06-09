package command

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
	"github.com/rs/zerolog/log"
)

func NewWalletCommand() *Command {
	return &Command{
		Command:      "wallet",
		ShortCommand: "w",
		Usage: `
Usage: clear [COMMAND]

This command cleans the terminal screen
		`,
		Help:             `Manage wallets`,
		Process:          Wallet_Process,
		AutoCompleteFunc: Wallet_AutoComplete,
	}
}

func Wallet_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}
	params, first_word := Params(input)

	re_subcommand := regexp.MustCompile(`^(\w*)$`)
	if m := re_subcommand.FindStringSubmatch(params); m != nil {
		si := m[1]

		is_subcommand := false
		for _, sc := range theme_subcommands {
			if sc == si {
				is_subcommand = true
				break
			}
		}

		if !is_subcommand {
			for _, sc := range []string{"open", "create"} {
				if input == "" || strings.Contains(sc, si) {
					options = append(options, ui.ACOption{Name: sc, Result: first_word + " " + sc + " "})
				}
			}
		}

		return "action", &options, input
	}

	re_demo := regexp.MustCompile(`^open\s+(\w*)$`)
	if m := re_demo.FindStringSubmatch(params); m != nil {
		t := m[1]

		// list all files in the wallet directory
		files, err := os.ReadDir(cmn.DataFolder + "/wallets")
		if err != nil {
			log.Error().Msgf("Error reading directory: %v", err)
		}

		for _, file := range files {
			if t == "" || strings.Contains(file.Name(), t) {
				options = append(options, ui.ACOption{Name: file.Name(), Result: first_word + " open " + file.Name()})
			}
		}

		return "file", &options, input
	}

	return input, &options, ""
}

func Wallet_Process(c *Command, input string) {
	//parse command subcommand parameters
	tokens := strings.Fields(input)
	if len(tokens) < 2 {
		fmt.Fprintln(ui.Terminal.Screen, c.Usage)
		return
	}
	//execute command
	subcommand := tokens[1]

	switch subcommand {
	case "open":
		wallet_name := "default"
		if len(tokens) >= 3 {
			wallet_name = tokens[2]
		}

		if err := wallet.OpenWallet(wallet_name); err != nil {
			ui.PrintErrorf("Error opening wallet: %v", err)
			return
		}

		ui.Printf("Wallet '%s' opened\n", wallet_name)
	case "create":

		// if len(tokens) < 3 {
		// 	ui.PrintErrorf("Please specify wallet name")
		// 	return
		// }

		// // check if file exists
		// if _, err := os.Stat(cmn.DataFolder + "/wallets/" + wallet_name + ".wallet"); err == nil {
		// 	ui.PrintErrorf("Wallet file already exists")
		// 	return
		// }

		ui.Gui.ShowPopup(ui.DlgWaletCreate())

	}
}
