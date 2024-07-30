package command

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/google/gousb"
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

	n := 1

	switch subcommand {
	case "list", "":
		ui.Printf("\nUsb Devices:\n")

		// Initialize a new Context.
		ctx := gousb.NewContext()
		defer ctx.Close()

		// Open all devices.
		devices, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {

			if cmn.IsSupportedDevice(uint16(desc.Vendor), uint16(desc.Product)) {
				return true
			}

			// ui.Printf("Device: VID:PID %04x:%04x, Bus %03d, Address %03d\n",
			// 	desc.Vendor, desc.Product, desc.Bus, desc.Address)

			// // The usbid package can be used to print out human readable information.
			// ui.Printf("%03d.%03d %s:%s %s\n", desc.Bus, desc.Address, desc.Vendor, desc.Product, usbid.Describe(desc))
			// ui.Printf("  Protocol: %s\n", usbid.Classify(desc))

			// // The configurations can be examined from the DeviceDesc, though they can only
			// // be set once the device is opened.  All configuration references must be closed,
			// // to free up the memory in libusb.
			// for _, cfg := range desc.Configs {
			// 	log.Debug().Msgf("Device: VID:PID %04x:%04x, Bus %03d, Address %03d\n", desc.Vendor, desc.Product, desc.Bus, desc.Address)

			// 	// This loop just uses more of the built-in and usbid pretty printing to list
			// 	// the USB devices.
			// 	ui.Printf("  %s:\n", cfg)
			// 	for _, intf := range cfg.Interfaces {
			// 		ui.Printf("    --------------\n")
			// 		for _, ifSetting := range intf.AltSettings {
			// 			ui.Printf("    %s\n", ifSetting)
			// 			ui.Printf("      %s\n", usbid.Classify(ifSetting))
			// 			for _, end := range ifSetting.Endpoints {
			// 				ui.Printf("      %s\n", end)
			// 			}
			// 		}
			// 	}
			// 	ui.Printf("    --------------\n")
			// }

			// After inspecting the descriptor, return true or false depending on whether
			// the device is "interesting" or not.  Any descriptor for which true is returned
			// opens a Device which is retuned in a slice (and must be subsequently closed).
			return false
		})
		if err != nil {
			ui.PrintErrorf("\nError listing usb devices: %v\n", err)
		}
		defer func() {
			for _, d := range devices {
				d.Close()
			}
		}()

		if len(devices) == 0 {
			ui.PrintErrorf("\nNo usb devices found\n")
			return
		}

		for _, dev := range devices {
			ui.Printf("%02d %-7s %s ", n, cmn.USBDeviceType(uint16(dev.Desc.Vendor), uint16(dev.Desc.Product)), dev.Desc.String())
			n++
		}

		ui.Printf("\n")

		ui.Flush()

	default:
		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
	}
}
