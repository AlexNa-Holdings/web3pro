package command

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
)

var signer_subcommands = []string{"remove", "add", "edit", "list"}

func NewSignerCommand() *Command {
	return &Command{
		Command:      "signer",
		ShortCommand: "s",
		Usage: `
Usage: signer [COMMAND]

Manage signers

Commands:
  add    - Add new signer
  list   - List signers
  remove - Remove signer  
		`,
		Help:             `Manage signers`,
		Process:          Signer_Process,
		AutoCompleteFunc: Signer_AutoComplete,
	}
}

func Signer_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}
	p := cmn.Split(input)
	command, subcommand, param := p[0], p[1], p[2]

	if !cmn.IsInArray(signer_subcommands, subcommand) {
		for _, sc := range signer_subcommands {
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	}

	if subcommand == "remove" || subcommand == "edit" {
		if wallet.CurrentWallet != nil {
			for _, signer := range wallet.CurrentWallet.SignersData {
				if cmn.Contains(signer.Name, param) {
					options = append(options, ui.ACOption{
						Name: signer.Name, Result: command + " " + subcommand + " '" + signer.Name + "'"})
				}
			}
		}
		return "signer", &options, subcommand
	}

	if subcommand == "add" {
		// TODO
	}

	return "", &options, ""
}

func Signer_Process(c *Command, input string) {
}
