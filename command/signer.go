package command

import (
	"strconv"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/signer"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/usb"
	"github.com/AlexNa-Holdings/web3pro/wallet"
)

var signer_subcommands = []string{"list", "remove", "promote", "add", "edit", "addresses"}

func NewSignerCommand() *Command {
	return &Command{
		Command:      "signer",
		ShortCommand: "s",
		Usage: `
Usage: signer [COMMAND]

Manage signers

Commands:
  add       - Add new signer
  list      - List signers
  remove    - Remove signer  
  edit      - Edit signer
  promote   - Promote copy to main signer
  addresses - List addresses of a signer
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

	if subcommand == "addresses" && param != "" {
		if wallet.CurrentWallet != nil {
			for _, s := range wallet.CurrentWallet.Signers {
				if s.Name == param {

					for d_id, d := range signer.STANDARD_DERIVATIONS {
						if cmn.Contains(d.Name, p3) {
							options = append(options, ui.ACOption{Name: d.Name, Result: command + " " + subcommand + " '" + param + "' " + d_id})
						}
					}
					return "derivation", &options, p3
				}
			}
		}
	}

	if subcommand == "remove" || subcommand == "edit" || subcommand == "addresses" {
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

	if subcommand == "promote" {
		if wallet.CurrentWallet != nil {
			for _, signer := range wallet.CurrentWallet.Signers {
				if signer.CopyOf != "" && cmn.Contains(signer.Name, param) {
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

	p := cmn.SplitN(input, 5)
	_, subcommand, p1, p2, p3 := p[0], p[1], p[2], p[3], p[4]

	switch subcommand {
	case "list", "":
		wallet.CurrentWallet.SortSigners()

		ui.Printf("List of signers:\n")

		for i, signer := range wallet.CurrentWallet.Signers {
			if signer.CopyOf == "" {
				ui.Printf("%-13s %-9s ", signer.Name, signer.Type)
			} else {
				if i == len(wallet.CurrentWallet.Signers)-1 || wallet.CurrentWallet.Signers[i+1].CopyOf != signer.CopyOf {
					ui.Printf("╰─ ")
				} else {
					ui.Printf("├─ ")
				}
				ui.Printf("%-10s %-9s ", signer.Name, signer.Type)
			}
			ui.Terminal.Screen.AddLink(gocui.ICON_EDIT, "command s edit '"+signer.Name+"'", "Edit signer '"+signer.Name+"'")
			ui.Printf(" ")
			ui.Terminal.Screen.AddLink(gocui.ICON_DELETE, "command s remove '"+signer.Name+"'", "Remove signer '"+signer.Name+"'")
			if signer.CopyOf != "" {
				ui.Printf(" ")
				ui.Terminal.Screen.AddLink(gocui.ICON_PROMOTE, "command s promote "+signer.Name, "Promote copy to main signer")
			}
			ui.Printf("\n")
		}

		ui.Printf("\n")
	case "addresses":
		path_format := ""

		if p2 == "" {
			path_format = signer.STANDARD_DERIVATIONS["default"].Format
		} else {
			if d, ok := signer.STANDARD_DERIVATIONS[p2]; ok {
				path_format = d.Format
			} else {
				if strings.Contains(p2, "%d") {
					path_format = p2
				} else {
					ui.PrintErrorf("Custom derivation path must contain %%d\n")
					return
				}
			}
		}

		from, _ := strconv.Atoi(p3)

		signer := wallet.CurrentWallet.GetSigner(p1)
		if signer == nil {
			ui.PrintErrorf("\nSigner not found\n")
			return
		}

		l, err := signer.GetAddresses(path_format, from, 10)
		if err != nil {
			ui.PrintErrorf("\nError getting addresses: %v\n", err)
			return
		}

		for i, s := range l {
			ui.Printf("%2d: %s ", from+i, s.Address.String())

			if ea := wallet.CurrentWallet.GetAddress(s.Address.String()); ea == nil {
				ui.Terminal.Screen.AddLink(gocui.ICON_ADD, "command address add "+
					s.Address.String()+
					" '"+p1+"' \""+s.Path+"\"", "Add address to wallet")
			} else {
				ui.Printf("%s ", ea.Name)
				ui.Terminal.Screen.AddLink(gocui.ICON_EDIT, "command address edit '"+ea.Name+"'", "Edit address")
				ui.Printf(" ")
				ui.Terminal.Screen.AddLink(gocui.ICON_DELETE, "command address remove '"+ea.Name+"'", "Remove address")
			}
			ui.Printf("\n")
		}
		ui.Printf("                                         ")
		ui.Terminal.Screen.AddLink("...more", "command signer addresses '"+p1+"' \""+path_format+"\" "+strconv.Itoa(from+10), "Show more addresses")
		ui.Printf("\n")
		ui.Terminal.Screen.ScrollBottom()

	case "add":
		ui.Gui.ShowPopup(ui.DlgSignerAdd(p1, p2))

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

	case "promote":
		s := wallet.CurrentWallet.GetSigner(p1)
		if s == nil {
			ui.PrintErrorf("Signer not found: %s\n", p1)
			return
		}

		if s.CopyOf == "" {
			ui.PrintErrorf("Signer %s is not a copy\n", p1)
			return
		}

		m := wallet.CurrentWallet.GetSigner(s.CopyOf)
		if m == nil {
			ui.PrintErrorf("Main signer not found: %s\n", s.CopyOf)
			return
		}

		ui.Gui.ShowPopup(ui.DlgConfirm(
			"Promote signer",
			`Are you sure you want to promote signer '`+p1+"' to main signer?\n",
			func() {
				// move all the addresses to the new main signer
				for _, a := range wallet.CurrentWallet.Addresses {
					if a.Signer == m.Name {
						a.Signer = p1
					}
				}

				m.CopyOf = s.Name
				s.CopyOf = ""
				// switch all the copies
				for _, signer := range wallet.CurrentWallet.Signers {
					if signer.CopyOf == m.Name {
						signer.CopyOf = p1
					}
				}

				err := wallet.CurrentWallet.Save()
				if err != nil {
					ui.PrintErrorf("\nFailed to save wallet: %s\n", err)
					return
				}

				ui.Printf("\nSigner %s promoted\n", p1)
			}))

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
