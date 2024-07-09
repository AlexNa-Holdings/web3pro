package command

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/rs/zerolog/log"
)

var blockchain_subcommands = []string{"remove", "add", "edit", "list", "use"}

func NewBlockchainCommand() *Command {
	return &Command{
		Command:      "blockchain",
		ShortCommand: "b",
		Usage: `
Usage: blockchain [COMMAND]

Manage blockchains

Commands:
  add [BLOCKCHAIN]    - Add new blockchain
  use [BLOCKCHAIN]    - Use blockchain
  list                - List blockchains
  remove [BLOCKCHAIN] - Remove blockchain  
		`,
		Help:             `Manage blockchains`,
		Process:          Blockchain_Process,
		AutoCompleteFunc: Blockchain_AutoComplete,
	}
}

func Blockchain_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}
	p := cmn.Split(input)
	command, subcommand, param := p[0], p[1], p[2]

	if !cmn.IsInArray(blockchain_subcommands, subcommand) {
		for _, sc := range blockchain_subcommands {
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	}

	if subcommand == "use" || subcommand == "remove" || subcommand == "edit" {
		if cmn.CurrentWallet != nil {
			for _, chain := range cmn.CurrentWallet.Blockchains {
				if cmn.Contains(chain.Name, param) {
					options = append(options, ui.ACOption{
						Name: chain.Name, Result: command + " " + subcommand + " '" + chain.Name + "'"})
				}
			}
		}
		return "blockchain", &options, subcommand
	}

	if subcommand == "add" && param != "" && strings.HasSuffix(input, " ") {
		return "", nil, ""
	}

	if subcommand == "add" {
		for _, chain := range cmn.PrefefinedBlockchains {
			if cmn.Contains(chain.Name, param) {
				options = append(options, ui.ACOption{Name: chain.Name, Result: command + " add '" + chain.Name + "' "})
			}
		}

		if param == "" || cmn.Contains("(custom)", param) {
			options = append(options, ui.ACOption{Name: "(custom)", Result: command + " add custom "})
		}

		return "blockchain", &options, param
	}

	return "", &options, ""
}

func Blockchain_Process(c *Command, input string) {

	if cmn.CurrentWallet == nil {
		ui.PrintErrorf("\nNo wallet open\n")
		return
	}

	//parse command subcommand parameters
	t := cmn.Split(input)
	subcommand, b_name := t[1], t[2]

	log.Debug().Msgf("Blockchain_Process: %s %s", subcommand, b_name)

	switch subcommand {
	case "add":
		if b_name == "" {
			ui.PrintErrorf("\nUsage: blockchain add [blockchain]\n")
			return
		}

		if b_name == "custom" {
			ui.Gui.ShowPopup(ui.DlgBlockchain(""))
		} else {
			// check if such blockchain already added
			for _, b := range cmn.CurrentWallet.Blockchains {
				if b.Name == b_name {
					ui.PrintErrorf("\nBlockchain %s already added\n", b_name)
					return
				}
			}

			for _, b := range cmn.PrefefinedBlockchains {
				if b.Name == b_name {

					bch := b
					cmn.CurrentWallet.Blockchains = append(cmn.CurrentWallet.Blockchains, &bch)

					cmn.CurrentWallet.AuditNativeTokens()
					err := cmn.CurrentWallet.Save()
					if err != nil {
						ui.PrintErrorf("\nFailed to save wallet: %s\n", err)
						return
					}

					ui.Printf("\nBlockchain %s added\n", b_name)

					ui.Printf(" Name: %s\n", b.Name)
					ui.Printf(" URL: %s\n", b.Url)
					ui.Printf(" Chain ID: %d\n", b.ChainId)
					ui.Printf(" Symbol: %s\n", b.Currency)
					ui.Printf(" Explorer: %s\n", b.ExplorerUrl)
					ui.Terminal.Screen.AddLink(gocui.ICON_EDIT, "command b edit '"+b.Name+"'", "Edit blockchain '"+b.Name+"'", "")

					ui.Printf("\n")
					return
				}
			}

			ui.PrintErrorf("\nBlockchain %s not found\n", b_name)
		}

	case "remove":
		if b_name == "" {
			ui.PrintErrorf("\nUsage: blockchain remove [blockchain]\n")
			return
		}

		for i, b := range cmn.CurrentWallet.Blockchains {
			if b.Name == b_name {

				ui.Gui.ShowPopup(ui.DlgConfirm(
					"Remove blockchain",
					`
<c>Are you sure you want to remove 
<c>blockchain '`+b_name+"' ?\n",
					func() {
						cmn.CurrentWallet.Blockchains = append(cmn.CurrentWallet.Blockchains[:i], cmn.CurrentWallet.Blockchains[i+1:]...)

						cmn.CurrentWallet.AuditNativeTokens()
						err := cmn.CurrentWallet.Save()
						if err != nil {
							ui.PrintErrorf("\nFailed to save wallet: %s\n", err)
							return
						}

						ui.Printf("\nBlockchain %s removed\n", b_name)
					}))
				return
			}
		}

		ui.PrintErrorf("\nBlockchain %s not found\n", b_name)

	case "list", "":
		ui.Printf("\nBlockchains:\n")

		for _, b := range cmn.CurrentWallet.Blockchains {
			ui.Terminal.Screen.AddLink(b.Name, "command b use '"+b.Name+"'", "Use blockchain '"+b.Name+"'", "")
			ui.Printf(" ")
			ui.Terminal.Screen.AddLink(gocui.ICON_EDIT, "command b edit '"+b.Name+"'", "Edit blockchain '"+b.Name+"'", "")
			ui.Terminal.Screen.AddLink(gocui.ICON_DELETE, "command b remove '"+b.Name+"'", "Remove blockchain '"+b.Name+"'", "")
			ui.Printf("\n")
		}

		ui.Printf("\n")
	case "edit":
		if b_name == "" {
			ui.PrintErrorf("\nUsage: blockchain edit [blockchain]\n")
			return
		}

		for _, b := range cmn.CurrentWallet.Blockchains {
			if b.Name == b_name {
				ui.Gui.ShowPopup(ui.DlgBlockchain(b_name))
				return
			}
		}

		ui.PrintErrorf("\nBlockchain %s not found\n", b_name)
	default:
		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
	}

}
