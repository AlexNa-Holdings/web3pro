package command

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/signer"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
)

var usb_subcommands = []string{"list"}

func NewUsbCommand() *Command {
	return &Command{
		Command:      "usb",
		ShortCommand: "",
		Usage: `
Usage: 

  usb  - List usb devices

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

	if !cmn.IsInArray(usb_subcommands, subcommand) {
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
	_, subcommand := p[0], p[1]

	switch subcommand {
	case "list", "":
		ui.Printf("\nUsb Devices:\n")

		l, err := cmn.Core.Enumerate()
		if err != nil {
			ui.PrintErrorf("\nError listing usb devices: %v\n", err)
			return
		}

		n := 1
		for _, u := range l {

			t := signer.GetType(u.Vendor, u.Product)
			device_name, err := signer.GetDeviceName(u)
			if err != nil {
				ui.PrintErrorf("\nError getting device name: %v\n", err)
				return
			}

			ui.Printf("%02d %-7s %s ", n, t, device_name)
			n++

			if wallet.CurrentWallet != nil {
				es := wallet.CurrentWallet.GetSigner(device_name)
				if es != nil {
					ui.Terminal.Screen.AddLink(gocui.ICON_EDIT, "command s edit '"+device_name+"'", "Edit signer")
					break
				} else {
					ui.Terminal.Screen.AddButton("Add signer", "command s add "+t+" '"+device_name+"'", "Add signer") //TODO
				}
			}
			ui.Printf("\n")
			ui.Printf("   vendor: %x\n", u.Vendor)
			ui.Printf("   product: %x\n", u.Product)
			ui.Printf("   path: %s\n", u.Path)
			ui.Printf("\n")
		}

		ui.Printf("\n")

		if len(l) == 0 {
			ui.Printf("No usb devices found\n")
		}

		ui.Flush()

	default:
		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
	}
}
