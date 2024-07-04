package command

import (
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
)

var token_subcommands = []string{"remove", "add", "edit", "list"}

func NewTokenCommand() *Command {
	return &Command{
		Command:      "token",
		ShortCommand: "t",
		Usage: `
Usage: token [COMMAND]

Manage tokens

Commands:
  add [BLOCKCHAIN] [ADDRESS]   - Add new token
  list                         - List tokens
  remove [BLOCKCHAIN] [ADDRESS]- Remove token  
		`,
		Help:             `Manage tokens`,
		Process:          Token_Process,
		AutoCompleteFunc: Token_AutoComplete,
	}
}

func Token_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}
	p := cmn.Split(input)
	command, subcommand, param := p[0], p[1], p[2]

	if !cmn.IsInArray(token_subcommands, subcommand) {
		for _, sc := range token_subcommands {
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	}

	if subcommand == "use" || subcommand == "remove" || subcommand == "edit" {
		if wallet.CurrentWallet != nil {
			for _, chain := range wallet.CurrentWallet.Blockchains {
				if cmn.Contains(chain.Name, param) {
					options = append(options, ui.ACOption{
						Name: chain.Name, Result: command + " " + subcommand + " '" + chain.Name + "'"})
				}
			}
		}
		return "blockchain", &options, subcommand
	}

	if subcommand == "add" {
		for _, chain := range cmn.PrefefinedBlockchains {
			if cmn.Contains(chain.Name, param) {
				options = append(options, ui.ACOption{Name: chain.Name, Result: command + " add '" + chain.Name + "' "})
			}
		}

		if param == "" || cmn.Contains("(custom)", param) {
			options = append(options, ui.ACOption{Name: "(custom)", Result: command + " add custom "})
		}

		return "blockchain", &options, param
	}

	return "", &options, ""
}

func Token_Process(c *Command, input string) {
	if wallet.CurrentWallet == nil {
		ui.PrintErrorf("\nNo wallet open\n")
		return
	}

	//parse command subcommand parameters
	tokens := cmn.Split(input)
	if len(tokens) < 2 {
		fmt.Fprintln(ui.Terminal.Screen, c.Usage)
		return
	}
	//execute command
	subcommand := tokens[1]

	switch subcommand {
	case "add":
	case "remove":
	case "list", "":

		ui.Printf("\nTokens:\n")

		// for _, b := range wallet.CurrentWallet.Blockchains {
		// 	ui.Terminal.Screen.AddLink(b.Name, "command b use "+b.Name, "Use blockchain '"+b.Name+"'", "")
		// 	ui.Printf(" ")
		// 	ui.Terminal.Screen.AddLink("\uf044", "command b edit "+b.Name, "Edit blockchain '"+b.Name+"'", "")
		// 	ui.Printf("\n")
		// }

		ui.Printf("\n")
	case "edit":
	default:
		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
	}

}
