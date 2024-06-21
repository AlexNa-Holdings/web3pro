package command

import (
	"strconv"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
)

var address_subcommands = []string{"remove", "add", "edit", "list", "use"}

func NewAddressCommand() *Command {
	return &Command{
		Command:      "address",
		ShortCommand: "a",
		Usage: `
Usage: address [COMMAND]

Manage addresses

Commands:
  add    - Add new address
  use    - Use address
  list   - List addresses
  remove - Remove address  
		`,
		Help:             `Manage addresses`,
		Process:          Address_Process,
		AutoCompleteFunc: Address_AutoComplete,
	}
}

func Address_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}
	p := cmn.Split(input)
	command, subcommand, param := p[0], p[1], p[2]

	if !cmn.IsInArray(blockchain_subcommands, subcommand) {
		for _, sc := range address_subcommands {
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	}

	if subcommand == "use" || subcommand == "remove" || subcommand == "edit" {
		if wallet.CurrentWallet != nil {
			for _, a := range wallet.CurrentWallet.Addresses {
				if cmn.Contains(a.Name+a.Address.String(), param) {
					options = append(options, ui.ACOption{
						Name:   a.Address.String() + " " + a.Name,
						Result: command + " " + subcommand + " '" + a.Name + "'"})
				}
			}
		}
		return "address", &options, subcommand
	}

	if subcommand == "add" {
		if wallet.CurrentWallet != nil {
			for _, s := range wallet.CurrentWallet.Signers {
				if cmn.Contains(s.Name, param) {
					options = append(options, ui.ACOption{
						Name:   s.Name,
						Result: command + " " + subcommand + " '" + s.Name + "'"})
				}
			}
			return "signer", &options, param
		}
	}

	return "", &options, ""
}

func Address_Process(c *Command, input string) {
	var err error

	if wallet.CurrentWallet == nil {
		ui.PrintErrorf("\nNo wallet open\n")
		return
	}

	//parse command subcommand parameters
	tokens := cmn.SplitN(input, 4)
	_, subcommand, p0, p1 := tokens[0], tokens[1], tokens[2], tokens[3]

	switch subcommand {
	case "add":
		if wallet.CurrentWallet == nil {
			ui.PrintErrorf("\nNo wallet open\n")
			return
		}

		if p0 == "" {
			ui.PrintErrorf("\nUsage: address add sigher\n")
			return
		}

		signer := wallet.CurrentWallet.GetSigner(p0)
		if signer == nil {
			ui.PrintErrorf("\nSigner not found\n")
			return
		}

		start_from := 0
		if p1 != "" {
			start_from, err = strconv.Atoi(p1)
			if err != nil || start_from < 0 {
				ui.PrintErrorf("\nInvalid start_from parameter: %s\n", p1)
				return
			}
		}

		l, err := signer.GetAddresses(start_from, 10)
		if err != nil {
			ui.PrintErrorf("\nError getting addresses: %v\n", err)
			return
		}

		for _, s := range l {
			ui.Printf("\nAddress: %s\n", s.Address.String())
		}

	case "remove":
	case "list":
	case "edit":
	default:
		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
	}

}
