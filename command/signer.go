package command

import (
	"strconv"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
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
  add [TYPE] [SERIAL]                    - Add new signer
  list                                   - List signers
  remove [SIGNER]                        - Remove signer
  edit [SIGNER]                          - Edit signer
  promote [SIGNER]                       - Promote signer to main signer
  addresses [SIGNER] [DERIVATION] [FROM] - List addresses
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
					for d_id, d := range cmn.STANDARD_DERIVATIONS {
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
			for _, s := range wallet.CurrentWallet.Signers {
				if cmn.Contains(s.Name, param) {
					options = append(options, ui.ACOption{
						Name: s.Name, Result: command + " " + subcommand + " '" + s.Name + "'"})
				}
			}
		}
		return "signer", &options, subcommand
	}

	if subcommand == "promote" {
		if wallet.CurrentWallet != nil {
			for _, s := range wallet.CurrentWallet.Signers {
				for _, c := range s.Copies {
					if cmn.Contains(c, param) {
						options = append(options, ui.ACOption{
							Name: c, Result: command + " " + subcommand + " '" + c + "'"})
					}
				}
			}
		}
		return "signer", &options, subcommand
	}

	if subcommand == "add" {
		if !cmn.IsInArray(cmn.KNOWN_SIGNER_TYPES, param) {
			for _, t := range cmn.KNOWN_SIGNER_TYPES {
				if cmn.Contains(t, param) {
					options = append(options, ui.ACOption{Name: t, Result: command + " add " + t + " "})
				}
			}
			return "type", &options, param
		}

		l, err := cmn.Core.Enumerate()
		if err != nil {
			ui.PrintErrorf("\nError listing usb devices: %v\n", err)
			return "", &options, ""
		}
		for _, u := range l {
			if param == cmn.GetDeviceType(u.Vendor, u.Product) {
				dn, err := cmn.GetDeviceName(u)
				if err == nil {
					if cmn.Contains(dn, p3) && p3 != dn {
						options = append(options, ui.ACOption{Name: dn, Result: command + " add " + param + " '" + dn + "'"})
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
		ui.Printf("List of signers:\n")

		for _, s := range wallet.CurrentWallet.Signers {
			ui.Printf("%-13s %-9s ", s.Name, s.Type)

			if s.Type == "mnemonics" {
				ui.Terminal.Screen.AddLink(gocui.ICON_EDIT, "command s edit '"+s.Name+"'", "Edit signer '"+s.Name+"'", "")
				ui.Printf(" ")
			}
			ui.Terminal.Screen.AddLink(gocui.ICON_DELETE, "command s remove '"+s.Name+"'", "Remove signer '"+s.Name+"'", "")
			ui.Printf("\n")
			for j, c := range s.Copies {
				if j != len(s.Copies)-1 {
					ui.Printf("├─ ")
				} else {
					ui.Printf("╰─ ")
				}
				ui.Printf("%-10s ", c)
				ui.Terminal.Screen.AddLink(gocui.ICON_DELETE, "command s remove '"+c+"'", "Remove signer '"+c+"'", "")
				ui.Printf(" ")
				ui.Terminal.Screen.AddLink(gocui.ICON_PROMOTE, "command s promote '"+c+"'", "Promote copy to main signer", "")
				ui.Printf("\n")
			}
		}

		ui.Printf("\n")
		ui.Flush()
	case "addresses":
		path_format := ""

		if p2 == "" {
			path_format = cmn.STANDARD_DERIVATIONS["default"].Format
		} else {
			if d, ok := cmn.STANDARD_DERIVATIONS[p2]; ok {
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

		s := wallet.CurrentWallet.GetSigner(p1)
		if s == nil {
			ui.PrintErrorf("\nSigner not found\n")
			return
		}

		l, err := s.GetAddresses(path_format, from, 10)
		if err != nil {
			ui.PrintErrorf("\nError getting addresses: %v\n", err)
			return
		}

		for i, s := range l {
			ui.Printf("%2d: ", from+i)
			ui.AddAddressLink(nil, &s.Address)
			ui.Printf(" ")

			if ea := wallet.CurrentWallet.GetAddress(s.Address.String()); ea == nil {
				ui.Terminal.Screen.AddLink(gocui.ICON_ADD, "command address add "+
					s.Address.String()+
					" '"+p1+"' \""+s.Path+"\"", "Add address to wallet", "")
			} else {
				ui.Printf("%s ", ea.Name)
				ui.Terminal.Screen.AddLink(gocui.ICON_EDIT, "command address edit '"+ea.Name+"'", "Edit address", "")
				ui.Printf(" ")
				ui.Terminal.Screen.AddLink(gocui.ICON_DELETE, "command address remove '"+ea.Name+"'", "Remove address", "")
			}
			ui.Printf("\n")
		}
		ui.Printf("                                         ")
		ui.Terminal.Screen.AddLink("...more", "command signer addresses '"+p1+"' \""+path_format+"\" "+strconv.Itoa(from+10), "Show more addresses", "")
		ui.Printf("\n")
		ui.Terminal.Screen.ScrollBottom()
		ui.Flush()

	case "add":
		ui.Gui.ShowPopup(ui.DlgSignerAdd(p1, p2))

	case "remove":
		for i, s := range wallet.CurrentWallet.Signers {
			if s.Name == p1 {
				ui.Gui.ShowPopup(ui.DlgConfirm(
					"Remove signer",
					`
<c> Are you sure you want to remove 
<c> signer '`+p1+"' ?\n",
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

			for j, c := range s.Copies {
				if c == p1 {
					ui.Gui.ShowPopup(ui.DlgConfirm(
						"Remove signer's copy",
						`
<c> Are you sure you want to remove 
<c> signer's copy '`+p1+"' ?\n",

						func() {
							s.Copies = append(s.Copies[:j], s.Copies[j+1:]...)

							err := wallet.CurrentWallet.Save()
							if err != nil {
								ui.PrintErrorf("\nFailed to save wallet: %s\n", err)
								return
							}
							ui.Printf("\nSigner's copy %s removed\n", p1)
						}))
					return
				}
			}
		}

		ui.PrintErrorf("Signer not found: %s\n", p1)

	case "promote":
		s, ci := wallet.CurrentWallet.GetSignerWithCopy(p1)
		if s == nil {
			ui.PrintErrorf("Signer not found: %s\n", p1)
			return
		}

		ui.Gui.ShowPopup(ui.DlgConfirm(
			"Promote signer",
			`Are you sure you want to promote signer '`+p1+"' to main signer?\n",
			func() {
				// move all the addresses to the new main signer
				for _, a := range wallet.CurrentWallet.Addresses {
					if a.Signer == s.Name {
						a.Signer = s.Copies[ci]
					}
				}

				//swap name and SN
				tmp_name := s.Name
				s.Name = s.Copies[ci]
				s.Copies[ci] = tmp_name

				err := wallet.CurrentWallet.Save()
				if err != nil {
					ui.PrintErrorf("\nFailed to save wallet: %s\n", err)
					return
				}

				ui.Printf("\nSigner %s promoted\n", p1)
			}))

	case "edit":
		for _, s := range wallet.CurrentWallet.Signers {
			if s.Name == p1 {
				ui.Gui.ShowPopup(ui.DlgSignerEdit(s.Name))
				return
			}
		}

		ui.PrintErrorf("Signer not found: %s\n", p1)
	default:
		ui.PrintErrorf("Unknown command: %s\n", subcommand)
	}
}
