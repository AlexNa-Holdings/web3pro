package command

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
)

var usb_subcommands = []string{"list"}

func NewUsbCommand() *Command {
	return &Command{
		Command:      "usb",
		ShortCommand: "",
		Subcommands:  usb_subcommands,
		Usage: `
Usage:

  usb  - List usb devices

		`,
		Help:             `List usb devices`,
		Process:          Usb_Process,
		AutoCompleteFunc: Usb_AutoComplete,
	}
}

func Usb_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}
	p := cmn.Split3(input)
	command, subcommand, _ := p[0], p[1], p[2]

	if !cmn.IsInArray(usb_subcommands, subcommand) {
		for _, sc := range usb_subcommands {
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	}

	return "", &options, ""
}

func Usb_Process(c *Command, input string) {
	p := cmn.Split3(input)
	_, subcommand := p[0], p[1]

	switch subcommand {
	case "list", "":
		ui.Printf("\nUsb Devices:\n")

		resp := bus.Fetch("usb", "list", nil)
		if resp.Error != nil {
			ui.PrintErrorf("Error listing usb devices: %v", resp.Error)
			return
		}

		l, ok := resp.Data.(bus.B_UsbList_Response)
		if !ok {
			ui.PrintErrorf("Error listing usb devices: %v", resp.Error)
			return
		}

		n := 1
		for _, u := range l {
			cs := " "
			if u.Connected {
				cs = "\U000f1616"
			}

			ui.Printf("%02d %s (%04x/%04x) %s (%s) path: %s\n", n, cs, u.VendorID, u.ProductID, u.Vendor, u.Product, u.Path)
			n++
		}

		ui.Printf("\n")

		if len(l) == 0 {
			ui.PrintErrorf("No usb devices found")
			return
		}

		ui.Printf("\n")

		ui.Flush()

	default:
		ui.PrintErrorf("Invalid subcommand: %s", subcommand)
	}
}
