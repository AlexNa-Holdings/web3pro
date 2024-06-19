package command

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
)

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
			for _, sc := range []string{"close", "create", "list", "open"} {
				if input == "" || strings.Contains(sc, si) {
					options = append(options, ui.ACOption{Name: sc, Result: first_word + " " + sc + " "})
				}
			}
		}

		return "action", &options, si
	}

	re_demo := regexp.MustCompile(`^open\s+(\w*)$`)
	if m := re_demo.FindStringSubmatch(params); m != nil {
		t := m[1]

		files := wallet.List()

		for _, file := range files {
			if t == "" || strings.Contains(file, t) {
				options = append(options, ui.ACOption{Name: file, Result: first_word + " open " + file})
			}
		}

		return "file", &options, t
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
			ui.Terminal.Screen.AddLink(file, "command w open "+file, "Open wallet "+file)
			ui.Printf("\n")
		}

		ui.Printf("\n")
	default:
		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
	}

}
