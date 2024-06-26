package command

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
	"github.com/ethereum/go-ethereum/common"
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
  add [ADDRESS] [SIGNER] [PATH] - Add new address
  use [ADDRESS]                 - Use address
  list                          - List addresses
  remove [ADDRESS]              - Remove address  
		`,
		Help:             `Manage addresses`,
		Process:          Address_Process,
		AutoCompleteFunc: Address_AutoComplete,
	}
}

func Address_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}

	if wallet.CurrentWallet == nil {
		return "", &options, ""
	}

	p := cmn.SplitN(input, 5)
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
		for _, a := range wallet.CurrentWallet.Addresses {
			if cmn.Contains(a.Name+a.Address.String(), param) {
				options = append(options, ui.ACOption{
					Name:   a.Address.String() + " " + a.Name,
					Result: command + " " + subcommand + " '" + a.Name + "'"})
			}
		}
		return "address", &options, subcommand
	}

	if subcommand == "add" {
		address, signer, _ := p[2], p[3], p[4]

		if common.IsHexAddress(address) {

			for _, s := range wallet.CurrentWallet.Signers {
				if s.IsConnected() && cmn.Contains(s.Name, signer) {
					options = append(options, ui.ACOption{
						Name:   s.Name,
						Result: command + " " + subcommand + " " + address + " '" + s.Name + "'"})
				}
			}
			return "signer", &options, param
		}
	}

	return "", &options, ""
}

func Address_Process(c *Command, input string) {

	if wallet.CurrentWallet == nil {
		ui.PrintErrorf("\nNo wallet open\n")
		return
	}

	//parse command subcommand parameters
	tokens := cmn.SplitN(input, 5)
	_, subcommand, p0, p1, p2 := tokens[0], tokens[1], tokens[2], tokens[3], tokens[4]

	switch subcommand {
	case "add":
		if wallet.CurrentWallet == nil {
			ui.PrintErrorf("\nNo wallet open\n")
			return
		}

		if !common.IsHexAddress(p0) {
			ui.PrintErrorf("\nInvalid address\n")
			return
		}

		signer := wallet.CurrentWallet.GetSigner(p1)
		if signer == nil {
			ui.PrintErrorf("\nSigner not found\n")
			return
		}
		ui.Gui.ShowPopup(ui.DlgAddressAdd(p0, p1, p2))

	case "remove":
		for i, a := range wallet.CurrentWallet.Addresses {
			if a.Name == p0 {
				ui.Gui.ShowPopup(ui.DlgConfirm(
					"Remove address",
					`
<c>Are you sure you want to remove address:
<c> `+a.Name+`
<c> `+a.Address.String()+"? \n",
					func() {
						wallet.CurrentWallet.Addresses = append(wallet.CurrentWallet.Addresses[:i], wallet.CurrentWallet.Addresses[i+1:]...)

						err := wallet.CurrentWallet.Save()
						if err != nil {
							ui.PrintErrorf("\nError saving wallet: %v\n", err)
							return
						}
						ui.Notification.Show("Address removed")
					}))

				return
			}
		}
		ui.PrintErrorf("\nAddress not found: %s\n", p0)
	case "list", "":
		ui.Printf("\nAddresses:\n")
		for _, a := range wallet.CurrentWallet.Addresses {
			ui.Printf("  ")
			ui.AddAddressLink(nil, &a.Address)
			ui.Printf("  %s %s ", a.Name, a.Signer)
			ui.Terminal.Screen.AddLink(gocui.ICON_EDIT, "command address edit '"+a.Name+"'", "Edit address")
			ui.Printf(" ")
			ui.Terminal.Screen.AddLink(gocui.ICON_DELETE, "command address remove '"+a.Name+"'", "Remove address")
			ui.Printf("\n")
		}

	case "edit":
		if wallet.CurrentWallet.GetAddressByName(p0) == nil {
			ui.PrintErrorf("\nAddress not found: %s\n", p0)
			return
		}
		ui.Gui.ShowPopup(ui.DlgAddressEdit(p0))
	default:
		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
	}

}
