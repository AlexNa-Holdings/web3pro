package command

import (
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/blockchain"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
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
  add    - Add new blockchain
  use    - Use blockchain
  list   - List blockchains
  remove - Remove blockchain  
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
		if wallet.CurrentWallet != nil {
			for _, chain := range wallet.CurrentWallet.Blockchains {
				if cmn.Contains(chain.Name, param) {
					options = append(options, ui.ACOption{
						Name: chain.Name, Result: command + " " + subcommand + " '" + chain.Name + "'"})
				}
			}
		}
		return "blockchain", &options, subcommand
	}

	if subcommand == "add" {
		for _, chain := range blockchain.PrefefinedBlockchains {
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
	//parse command subcommand parameters
	tokens := cmn.Split(input)
	if len(tokens) < 2 {
		fmt.Fprintln(ui.Terminal.Screen, c.Usage)
		return
	}
	//execute command
	subcommand := tokens[1]

	switch subcommand {
	case "add":
		if wallet.CurrentWallet == nil {
			ui.PrintErrorf("\nNo wallet open\n")
			return
		}

		if len(tokens) < 3 {
			ui.PrintErrorf("\nUsage: blockchain add [blockchain]\n")
			return
		}

		blockchain_name := tokens[2]

		if blockchain_name == "custom" {
			// TODO
		} else {
			// check if such blockchain already added
			for _, b := range wallet.CurrentWallet.Blockchains {
				if b.Name == blockchain_name {
					ui.PrintErrorf("\nBlockchain %s already added\n", blockchain_name)
					return
				}
			}

			for _, b := range blockchain.PrefefinedBlockchains {
				if b.Name == blockchain_name {
					wallet.CurrentWallet.Blockchains = append(wallet.CurrentWallet.Blockchains, b)

					err := wallet.CurrentWallet.Save()
					if err != nil {
						ui.PrintErrorf("\nFailed to save wallet: %s\n", err)
						return
					}

					ui.Printf("\nBlockchain %s added\n", blockchain_name)

					ui.Printf(" Name: %s\n", b.Name)
					ui.Printf(" URL: %s\n", b.Url)
					ui.Printf(" Chain ID: %d\n", b.ChainId)
					ui.Printf(" Symbol: %s\n", b.Currency)
					ui.Printf(" Explorer: %s\n", b.ExplorerUrl)
					ui.Terminal.Screen.AddLink(ui.ICON_EDIT, "command b edit '"+b.Name+"'", "Edit blockchain '"+b.Name+"'")

					ui.Printf("\n")
					return
				}
			}

			ui.PrintErrorf("\nBlockchain %s not found\n", blockchain_name)
		}

	case "remove":
		if wallet.CurrentWallet == nil {
			ui.PrintErrorf("\nNo wallet open\n")
			return
		}

		if len(tokens) < 3 {
			ui.PrintErrorf("\nUsage: blockchain remove [blockchain]\n")
			return
		}

		blockchain_name := tokens[2]

		for i, b := range wallet.CurrentWallet.Blockchains {
			if b.Name == blockchain_name {

				ui.Gui.ShowPopup(ui.DlgConfirm(
					"Remove blockchain",
					`
<c>Are you sure you want to remove 
<c>blockchain '`+blockchain_name+"' ?\n",
					func() {
						wallet.CurrentWallet.Blockchains = append(wallet.CurrentWallet.Blockchains[:i], wallet.CurrentWallet.Blockchains[i+1:]...)

						err := wallet.CurrentWallet.Save()
						if err != nil {
							ui.PrintErrorf("\nFailed to save wallet: %s\n", err)
							return
						}

						ui.Printf("\nBlockchain %s removed\n", blockchain_name)
					}))
				return
			}
		}

		ui.PrintErrorf("\nBlockchain %s not found\n", blockchain_name)

	case "list":
		if wallet.CurrentWallet == nil {
			ui.PrintErrorf("\nNo wallet open\n")
			return
		}

		ui.Printf("\nBlockchains:\n")

		for _, b := range wallet.CurrentWallet.Blockchains {
			ui.Terminal.Screen.AddLink(b.Name, "command b use "+b.Name, "Use blockchain '"+b.Name+"'")
			ui.Printf(" ")
			ui.Terminal.Screen.AddLink("\uf044", "command b edit "+b.Name, "Edit blockchain '"+b.Name+"'")
			ui.Printf("\n")
		}

		ui.Printf("\n")
	case "edit":
		if wallet.CurrentWallet == nil {
			ui.PrintErrorf("\nNo wallet open\n")
			return
		}

		if len(tokens) < 3 {
			ui.PrintErrorf("\nUsage: blockchain edit [blockchain]\n")
			return
		}

		blockchain_name := tokens[2]

		for _, b := range wallet.CurrentWallet.Blockchains {
			if b.Name == blockchain_name {
				ui.Gui.ShowPopup(ui.DlgBlockchainEdit(blockchain_name))
				return
			}
		}

		ui.PrintErrorf("\nBlockchain %s not found\n", blockchain_name)
	default:
		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
	}

}
