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
	p := cmn.Split(input)
	command, subcommand, _ := p[0], p[1], p[2]

	if !cmn.IsInArray(usb_subcommands, subcommand) {
		for _, sc := range usb_subcommands {
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

// func Usb_Process(c *Command, input string) {
// 	p := cmn.Split(input)
// 	_, subcommand := p[0], p[1]

// 	switch subcommand {
// 	case "list", "":
// 		ui.Printf("\nUsb Devices:\n")

// 		l, err := cmn.Core.Enumerate()
// 		if err != nil {
// 			ui.PrintErrorf("\nError listing usb devices: %v\n", err)
// 			return
// 		}

// 		n := 1
// 		for _, u := range l {

// 			t := cmn.GetDeviceType(u.Vendor, u.Product)

// 			log.Trace().Msgf("Device type: %s", t)

// 			device_name, err := cmn.GetDeviceName(u)
// 			if err != nil {
// 				ui.PrintErrorf("\nError getting device name: %v\n", err)
// 				return
// 			}

// 			ui.Printf("%02d %-7s %s ", n, t, device_name)
// 			n++

// 			if cmn.CurrentWallet != nil {
// 				es := cmn.CurrentWallet.GetSigner(device_name)
// 				if es != nil {
// 					ui.Terminal.Screen.AddLink(gocui.ICON_EDIT, "command s edit '"+device_name+"'", "Edit signer", "")
// 				} else {
// 					ui.Terminal.Screen.AddButton("Add signer", "command s add "+t+" '"+device_name+"'", "Add signer", "", "", "")
// 				}
// 			}
// 			ui.Printf("\n")
// 			switch t {
// 			// case "ledger":
// 			case "trezor":
// 				ui.Printf(cmn.WalletTrezorDriver.PrintDetails(u.Path))
// 			}
// 		}

// 		ui.Printf("\n")

// 		if len(l) == 0 {
// 			ui.Printf("No usb devices found\n")
// 		}

// 		ui.Flush()

// 	default:
// 		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
// 	}
// }

func Usb_Process(c *Command, input string) {
	p := cmn.Split(input)
	_, subcommand := p[0], p[1]

	switch subcommand {
	case "list", "":
		ui.Printf("\nUsb Devices:\n")

		resp := bus.Fetch("usb", "list", nil)
		if resp.Error != nil {
			ui.PrintErrorf("\nError listing usb devices: %v\n", resp.Error)
			return
		}

		l, ok := resp.Data.(bus.B_UsbList_Response)
		if !ok {
			ui.PrintErrorf("\nError listing usb devices: %v\n", resp.Error)
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
			ui.PrintErrorf("\nNo usb devices found\n")
			return
		}

		ui.Printf("\n")

		ui.Flush()

	default:
		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
	}
}
