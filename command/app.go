package command

import (
	"sort"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
)

var app_subcommands = []string{"on", "off", "remove", "list", "add_addr", "remove_addr", "promote_addr"}

func NewAppCommand() *Command {
	return &Command{
		Command:      "application",
		ShortCommand: "app",
		Usage: `
Usage: application [COMMAND]

Manage web applications (origins)

Commands:
  list [URL]                - List web applications
  remove [URL]              - Remove address  
  remove_addr [URL] [ADDR] 	- Remove address access
  add_addr [URL] [ADDR]     - Add address access
  promote_addr [URL] [ADDR] - Promote address
  on                        - Open application window
  off                       - Close application window
		`,
		Help:             `Manage connected web applications`,
		Process:          App_Process,
		AutoCompleteFunc: App_AutoComplete,
	}
}

func App_AutoComplete(input string) (string, *[]ui.ACOption, string) {

	if cmn.CurrentWallet == nil {
		return "", nil, ""
	}

	w := cmn.CurrentWallet

	options := []ui.ACOption{}

	p := cmn.SplitN(input, 4)
	command, subcommand, origin, addr := p[0], p[1], p[2], p[3]

	if !cmn.IsInArray(app_subcommands, subcommand) {
		for _, sc := range app_subcommands {
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	}

	if subcommand == "add_addr" {
		o := w.GetOrigin(origin)
		if o != nil {
			for _, a := range w.Addresses {
				if o.IsAllowed(a.Address) {
					continue
				}

				if !cmn.Contains(a.Address.String()+a.Name, addr) {
					continue
				}

				options = append(options, ui.ACOption{
					Name:   cmn.ShortAddress(a.Address) + " " + a.Name,
					Result: "app add_addr '" + origin + "' '" + a.Address.String() + "'"})
			}
			return "address", &options, addr
		}
	}

	if subcommand == "remove_addr" || subcommand == "promote_addr" {
		o := w.GetOrigin(origin)
		if o != nil {
			for i, na := range o.Addresses {

				if i == 0 && subcommand == "promote_addr" {
					continue
				}

				a := w.GetAddress(na.Hex())
				if a == nil {
					continue
				}

				if !cmn.Contains(a.Address.String()+a.Name, addr) {
					continue
				}

				options = append(options, ui.ACOption{
					Name:   cmn.ShortAddress(a.Address) + " " + a.Name,
					Result: "app " + subcommand + " '" + origin + "' '" + a.Name + "'"})
			}
			return "address", &options, addr
		}
	}

	if subcommand == "remove" || subcommand == "list" || subcommand == "add_addr" ||
		subcommand == "remove_addr" || subcommand == "promote_addr" {
		for _, o := range w.Origins {
			if cmn.Contains(o.URL, origin) {
				options = append(options, ui.ACOption{
					Name:   o.URL,
					Result: command + " " + subcommand + " '" + o.URL + "'"})
			}
		}
		return "application", &options, origin
	}

	return "", &options, ""
}

func App_Process(c *Command, input string) {
	if cmn.CurrentWallet == nil {
		ui.PrintErrorf("\nNo wallet open\n")
		return
	}

	w := cmn.CurrentWallet

	p := cmn.SplitN(input, 4)
	_, subcommand, origin, addr := p[0], p[1], p[2], p[3]

	switch subcommand {
	case "list", "":
		ui.Printf("Connected web applications:\n\n")

		sort.Slice(w.Origins, func(i, j int) bool {
			return w.Origins[i].URL < w.Origins[j].URL
		})

		for _, o := range w.Origins {
			ui.Printf("%s\n", o.URL)

			if p[2] != "" && o.URL != p[2] {
				continue
			}

			for i, na := range o.Addresses {
				a := w.GetAddress(na.Hex())
				if a == nil {
					a = &cmn.Address{
						Address: na,
						Name:    "Unknown",
					}
				}

				if i == 0 {
					ui.Printf("  *%d ", i)
				} else {
					ui.Printf("   %d ", i)
				}

				ui.AddAddressShortLink(nil, a.Address)
				ui.Printf(" ")
				ui.Terminal.Screen.AddLink(gocui.ICON_DELETE,
					"command app remove_addr '"+o.URL+"' '"+a.Name+"'",
					"Remove access for the address", "")

				if i == 0 {
					ui.Printf("  ")
				} else {
					ui.Terminal.Screen.AddLink(gocui.ICON_PROMOTE,
						"command app promote_addr '"+o.URL+"' '"+a.Name+"'",
						"Promote the address", "")
				}

				ui.Printf("%-14s (%s) \n", a.Name, a.Signer)

			}
		}
	case "remove":
		err := w.RemoveOrigin(origin)
		if err != nil {
			ui.PrintErrorf("Error removing origin: %v\n", err)
			return
		}
		err = w.Save()
		if err != nil {
			ui.PrintErrorf("Error saving wallet: %v\n", err)
			return
		}
		bus.Send("ui", "notify", "Web application removed: "+origin)
	case "remove_addr":
		w.RemoveOriginAddress(origin, addr)
		err := w.Save()
		if err != nil {
			ui.PrintErrorf("Error saving wallet: %v\n", err)
			return
		}
		bus.Send("ui", "notify", "Address removed: "+addr)
	case "add_addr":
		w.AddOriginAddress(origin, addr)
		err := w.Save()
		if err != nil {
			ui.PrintErrorf("Error saving wallet: %v\n", err)
			return
		}
		bus.Send("ui", "notify", "Address added: "+addr)
	case "promote_addr":
		w.PromoteOriginAddress(origin, addr)
		err := w.Save()
		if err != nil {
			ui.PrintErrorf("Error saving wallet: %v\n", err)
			return
		}
		bus.Send("ui", "notify", "Address promoted: "+addr)
	case "on":
		w.AppsPaneOn = true
		err := w.Save()
		if err != nil {
			ui.PrintErrorf("Error saving wallet: %v\n", err)
			return
		}
	case "off":
		w.AppsPaneOn = false
		err := w.Save()
		if err != nil {
			ui.PrintErrorf("Error saving wallet: %v\n", err)
			return
		}

	default:
		ui.PrintErrorf("Unknown command: %s\n", subcommand)
	}

}
