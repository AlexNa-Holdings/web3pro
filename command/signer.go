package command

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/signer"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/usb"
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
	p := cmn.SplitN(input, 4)
	command, subcommand, param, p3 := p[0], p[1], p[2], p[3]

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
			for _, signer := range wallet.CurrentWallet.Signers {
				if cmn.Contains(signer.Name, param) {
					options = append(options, ui.ACOption{
						Name: signer.Name, Result: command + " " + subcommand + " '" + signer.Name + "'"})
				}
			}
		}
		return "signer", &options, subcommand
	}

	if subcommand == "add" {
		if !cmn.IsInArray(signer.KNOWN_SIGNER_TYPES, param) {
			for _, t := range signer.KNOWN_SIGNER_TYPES {
				if cmn.Contains(t, param) {
					options = append(options, ui.ACOption{Name: t, Result: command + " add " + t + " "})
				}
			}
			return "type", &options, param
		}

		l, _ := usb.List() // ignore error
		for _, u := range l {
			if param == signer.GetType(u.Manufacturer, u.Product) {
				sn, err := usb.GetSN(u)
				if err == nil {
					if cmn.Contains(sn, p3) && p3 != sn {
						options = append(options, ui.ACOption{Name: sn, Result: command + " add " + param + " " + sn})
					}
				}
			}
		}
		return "Serial number", &options, ""
	}

	return "", &options, ""
}

func Signer_Process(c *Command, input string) {

	if wallet.CurrentWallet == nil {
		ui.PrintErrorf("No wallet opened\n")
		return
	}

	p := cmn.SplitN(input, 4)
	_, subcommand, p1, p2 := p[0], p[1], p[2], p[3]

	switch subcommand {
	case "list", "":
		for _, signer := range wallet.CurrentWallet.Signers {
			ui.Printf("%-10s %s ", signer.Type, signer.Name)
			ui.Terminal.Screen.AddLink(gocui.ICON_EDIT, "command s edit '"+signer.Name+"'", "Edit signer '"+signer.Name+"'")
			ui.Printf(" ")
			ui.Terminal.Screen.AddLink(gocui.ICON_DELETE, "command s remove '"+signer.Name+"'", "Remove signer '"+signer.Name+"'")
			ui.Printf("\n")
		}

		ui.Printf("\n")

	case "add":
		ui.Gui.ShowPopup(ui.DlgSignerCreate(p1, p2))

	case "remove":
		for i, signer := range wallet.CurrentWallet.Signers {
			if signer.Name == p1 {

				ui.Gui.ShowPopup(ui.DlgConfirm(
					"Remove signer",
					`
<c>Are you sure you want to remove 
<c>signer '`+p1+"' ?\n",
					func() {
						wallet.CurrentWallet.Signers = append(wallet.CurrentWallet.Signers[:i], wallet.CurrentWallet.Signers[i+1:]...)

						err := wallet.CurrentWallet.Save()
						if err != nil {
							ui.PrintErrorf("\nFailed to save wallet: %s\n", err)
							return
						}

						ui.Printf("\nSigner %s removed\n", p1)
					}))
				return
			}
		}

	case "edit":
		for _, signer := range wallet.CurrentWallet.Signers {
			if signer.Name == p1 {
				ui.Gui.ShowPopup(ui.DlgSignerEdit(signer.Name))
				return
			}
		}

		ui.PrintErrorf("Signer not found: %s\n", p1)
	default:
		ui.PrintErrorf("Unknown command: %s\n", subcommand)
	}
}