package command

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/signer"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/usb_support"
	"github.com/AlexNa-Holdings/web3pro/wallet"
)

var usb_subcommands = []string{"list"}

func NewUsbCommand() *Command {
	return &Command{
		Command:      "usb",
		ShortCommand: "",
		Usage: `
Usage: usb [COMMAND]

Manage usb

Commands:
  list   - List usb devices
		`,
		Help:             `Manage usb devices`,
		Process:          Usb_Process,
		AutoCompleteFunc: Usb_AutoComplete,
	}
}

func Usb_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}
	p := cmn.Split(input)
	command, subcommand, _ := p[0], p[1], p[2]

	if !cmn.IsInArray(blockchain_subcommands, subcommand) {
		for _, sc := range blockchain_subcommands {
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	}

	if subcommand == "list" {
		options = append(options, ui.ACOption{Name: "all", Result: command + " " + subcommand + " all"})
	}

	return "", &options, ""
}

func Usb_Process(c *Command, input string) {
	p := cmn.Split(input)
	_, subcommand, param := p[0], p[1], p[2]

	switch subcommand {
	case "list", "":
		ui.Printf("\nUsb Devices:\n")

		l, err := usb_support.List()
		if err != nil {
			ui.PrintErrorf("\nError listing usb devices: %v\n", err)
			return
		}

		n := 1
		for _, u := range l {

			sn, err := usb_support.GetSN(u)
			if err != nil {
				ui.PrintErrorf("\nError getting usb serial number: %v\n", err)
				return
			}

			if param != "all" && signer.GetType(u.Manufacturer, u.Product) == "" {
				continue
			}

			ui.Printf("%02d ", n)
			n++

			if wallet.CurrentWallet != nil {
				es := wallet.CurrentWallet.GetSignerByTypeAndSN(signer.GetType(u.Manufacturer, u.Product), sn)
				if es != nil {
					ui.Printf("'" + es.Name + "' ")
					ui.Terminal.Screen.AddLink(gocui.ICON_EDIT, "command s edit '"+es.Name+"'", "Edit signer")
					break
				} else {
					t := signer.GetType(u.Manufacturer, u.Product)
					if t != "" {
						ui.Terminal.Screen.AddButton("Add signer", "command s add "+t+" "+u.Serial, "Add signer")
					}
				}
			}
			ui.Printf("\n")
			ui.Printf("   manufacturer: %s\n", u.Manufacturer)
			ui.Printf("   product: %s\n", u.Product)
			ui.Printf("   serial: %s\n", sn)
			ui.Printf("\n")
		}

	default:
		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
	}
}
