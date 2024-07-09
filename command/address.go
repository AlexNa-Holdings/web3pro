package command

import (
	"sort"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
)

var address_subcommands = []string{"remove", "edit", "list", "use"}

func NewAddressCommand() *Command {
	return &Command{
		Command:      "address",
		ShortCommand: "a",
		Usage: `
Usage: address [COMMAND]

Manage addresses

Commands:
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

	if cmn.CurrentWallet == nil {
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
		for _, a := range cmn.CurrentWallet.Addresses {
			if cmn.Contains(a.Name+a.Address.String(), param) {
				options = append(options, ui.ACOption{
					Name:   cmn.ShortAddress(a.Address) + " " + a.Name,
					Result: command + " " + subcommand + " '" + a.Name + "'"})
			}
		}
		return "address", &options, subcommand
	}

	return "", &options, ""
}

func Address_Process(c *Command, input string) {

	if cmn.CurrentWallet == nil {
		ui.PrintErrorf("\nNo wallet open\n")
		return
	}

	//parse command subcommand parameters
	tokens := cmn.SplitN(input, 5)
	_, subcommand, p0 := tokens[0], tokens[1], tokens[2]

	switch subcommand {
	case "remove":
		for i, a := range cmn.CurrentWallet.Addresses {
			if a.Name == p0 {
				ui.Gui.ShowPopup(ui.DlgConfirm(
					"Remove address",
					`
<c>Are you sure you want to remove address:
<c> `+a.Name+`
<c> `+a.Address.String()+"? \n",
					func() {
						cmn.CurrentWallet.Addresses = append(cmn.CurrentWallet.Addresses[:i], cmn.CurrentWallet.Addresses[i+1:]...)

						err := cmn.CurrentWallet.Save()
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

		sort.Slice(cmn.CurrentWallet.Addresses, func(i, j int) bool {
			return cmn.CurrentWallet.Addresses[i].Name < cmn.CurrentWallet.Addresses[j].Name
		})
		ui.Printf("\nAddresses:\n")
		for _, a := range cmn.CurrentWallet.Addresses {
			ui.AddAddressShortLink(nil, a.Address)
			ui.Printf(" ")
			ui.Terminal.Screen.AddLink(gocui.ICON_EDIT, "command address edit '"+a.Name+"'", "Edit address", "")
			ui.Printf(" ")
			ui.Terminal.Screen.AddLink(gocui.ICON_DELETE, "command address remove '"+a.Name+"'", "Remove address", "")
			ui.Printf("  %-14s (%s) \n", a.Name, a.Signer)
		}

	case "edit":
		if cmn.CurrentWallet.GetAddressByName(p0) == nil {
			ui.PrintErrorf("\nAddress not found: %s\n", p0)
			return
		}
		ui.Gui.ShowPopup(ui.DlgAddressEdit(p0))
	default:
		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
	}

}
