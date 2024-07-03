package command

import (
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
)

var wallet_subcommands = []string{"close", "create", "list", "open"}

func NewWalletCommand() *Command {
	return &Command{
		Command:      "wallet",
		ShortCommand: "w",
		Usage: `
Usage: wallet [COMMAND]

Manage wallets

Commands:
  open <wallet>  Open wallet
  create         Create new wallet
  close          Close current wallet
  list           List wallets

		`,
		Help:             `Manage wallets`,
		Process:          Wallet_Process,
		AutoCompleteFunc: Wallet_AutoComplete,
	}
}

func Wallet_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}
	p := cmn.Split(input)
	command, subcommand, param := p[0], p[1], p[2]

	if !cmn.IsInArray(wallet_subcommands, subcommand) {
		for _, sc := range wallet_subcommands {
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	}

	if subcommand == "open" {
		files := wallet.List()

		for _, file := range files {
			if param == "" || strings.Contains(file, param) {
				options = append(options, ui.ACOption{Name: file, Result: command + " open " + file})
			}
		}

		return "file", &options, param
	}

	return "", &options, ""
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
		if len(tokens) != 3 {
			ui.PrintErrorf("Please specify wallet name")
			return
		}
		ui.Gui.ShowPopup(ui.DlgWaletOpen(tokens[2]))

	case "create":
		ui.Gui.ShowPopup(ui.DlgWaletCreate())
	case "close":
		if wallet.CurrentWallet != nil {
			wallet.CurrentWallet = nil
			ui.Terminal.SetCommandPrefix(ui.DEFAULT_COMMAND_PREFIX)
			ui.Notification.Show("Wallet closed")
		} else {
			ui.PrintErrorf("No wallet open")
		}
	case "list":
		files := wallet.List()
		if files == nil {
			ui.PrintErrorf("Error reading directory")
			return
		}

		ui.Printf("\nWallets:\n")

		for _, file := range files {
			ui.Terminal.Screen.AddLink(file, "command w open "+file, "Open wallet "+file, "")
			ui.Printf("\n")
		}

		ui.Printf("\n")
	default:
		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
	}

}
