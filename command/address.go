package command

import (
	"sort"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/ethereum/go-ethereum/common"
)

var address_subcommands = []string{"remove", "edit", "list", "set", "add"}

func NewAddressCommand() *Command {
	return &Command{
		Command:      "address",
		ShortCommand: "a",
		Subcommands:  address_subcommands,
		Usage: `
Usage: address [COMMAND]

Manage addresses

Commands:
  add [ADDRESS] [SIGNER] [PATH] - Add new address
  set [ADDRESS]                 - set the address
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

	w := cmn.CurrentWallet

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

	if subcommand == "set" || subcommand == "remove" || subcommand == "edit" {
		for _, a := range w.Addresses {
			if cmn.Contains(a.Name+a.Address.String(), param) {
				options = append(options, ui.ACOption{
					Name:   cmn.ShortAddress(a.Address) + " " + a.Name,
					Result: command + " " + subcommand + " '" + a.Name + "'"})
			}
		}
		return "address", &options, param
	}

	if subcommand == "add" {
		address, signer, _ := p[2], p[3], p[4]

		if common.IsHexAddress(address) {

			for _, s := range w.Signers {
				if cmn.Contains(s.Name, signer) {
					options = append(options, ui.ACOption{
						Name:   s.Name,
						Result: command + " " + subcommand + " " + address + " '" + s.Name + "'"})
				}
			}
			return "signer", &options, signer
		}
	}

	return "", &options, ""
}

func Address_Process(c *Command, input string) {
	if cmn.CurrentWallet == nil {
		ui.PrintErrorf("No wallet open")
		return
	}

	w := cmn.CurrentWallet

	//parse command subcommand parameters
	tokens := cmn.SplitN(input, 5)
	_, subcommand, p0, p1, p2 := tokens[0], tokens[1], tokens[2], tokens[3], tokens[4]

	switch subcommand {
	case "add":
		if !common.IsHexAddress(p0) {
			ui.PrintErrorf("Invalid address")
			return
		}

		signer := w.GetSigner(p1)
		if signer == nil {
			ui.PrintErrorf("Signer not found")
			return
		}
		bus.Send("ui", "popup", ui.DlgAddressAdd(p0, p1, p2))
	case "remove":
		for i, a := range w.Addresses {
			if a.Name == p0 {
				bus.Send("ui", "popup", ui.DlgConfirm(
					"Remove address",
					`
<c>Are you sure you want to remove address:
<c> `+a.Name+`
<c> `+a.Address.String()+"? \n",
					func() bool {
						w.Addresses = append(w.Addresses[:i], w.Addresses[i+1:]...)

						err := w.Save()
						if err != nil {
							ui.PrintErrorf("Error saving wallet: %v", err)
							return false
						}
						ui.Notification.Show("Address removed")
						return true
					}))

				return
			}
		}
		ui.PrintErrorf("Address not found: %s", p0)
	case "list", "":

		sort.Slice(w.Addresses, func(i, j int) bool {
			return w.Addresses[i].Name < w.Addresses[j].Name
		})
		ui.Printf("\nAddresses:\n")
		for _, a := range w.Addresses {
			cmn.AddAddressShortLink(ui.Terminal.Screen, a.Address)
			ui.Printf(" ")
			ui.Terminal.Screen.AddLink(cmn.ICON_EDIT, "command address edit '"+a.Name+"'", "Edit address", "")
			ui.Terminal.Screen.AddLink(cmn.ICON_DELETE, "command address remove '"+a.Name+"'", "Remove address", "")
			ui.Printf(" %-14s (%s) \n", a.Name, a.Signer)
		}

	case "edit":
		if w.GetAddressByName(p0) == nil {
			ui.PrintErrorf("Address not found: %s", p0)
			return
		}
		bus.Send("ui", "popup", ui.DlgAddressEdit(p0))
	case "set":
		fa := w.GetAddressByName(p0)
		if fa == nil {
			ui.PrintErrorf("Address not found: %s", p0)
			return
		}
		w.CurrentAddress = fa.Address
		err := w.Save()
		if err != nil {
			ui.PrintErrorf("Error saving wallet: %v", err)
			return
		}

	default:
		ui.PrintErrorf("Invalid subcommand: %s", subcommand)
	}

}
