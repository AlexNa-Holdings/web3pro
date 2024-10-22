package command

import (
	"sort"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
)

var blockchain_subcommands = []string{"remove", "add", "edit", "list", "set"}

func NewBlockchainCommand() *Command {
	return &Command{
		Command:      "blockchain",
		ShortCommand: "b",
		Usage: `
Usage: blockchain [COMMAND]

Manage blockchains

Commands:
  add [BLOCKCHAIN]    - Add new blockchain
  set [BLOCKCHAIN]    - Set blockchain
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

	if subcommand == "set" || subcommand == "remove" || subcommand == "edit" {
		if cmn.CurrentWallet != nil {
			for _, chain := range cmn.CurrentWallet.Blockchains {
				if cmn.Contains(chain.Name, param) {
					options = append(options, ui.ACOption{
						Name: chain.Name, Result: command + " " + subcommand + " '" + chain.Name + "'"})
				}
			}
		}
		return "blockchain", &options, param
	}

	if subcommand == "add" && param != "" && strings.HasSuffix(input, " ") {
		return "", nil, ""
	}

	if subcommand == "add" {
		for _, chain := range cmn.PredefinedBlockchains {
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
		ui.PrintErrorf("No wallet open")
		return
	}

	w := cmn.CurrentWallet

	//parse command subcommand parameters
	t := cmn.Split(input)
	subcommand, b_name := t[1], t[2]

	switch subcommand {
	case "add":
		if b_name == "custom" || b_name == "" {
			bus.Send("ui", "popup", ui.DlgBlockchain(0))
		} else {

			// check if such blockchain already added
			for _, b := range w.Blockchains {
				if b.Name == b_name {
					ui.PrintErrorf("Blockchain %s already added", b_name)
					return
				}
			}

			for _, b := range cmn.PredefinedBlockchains {
				if b.Name == b_name {

					bch := b
					err := w.AddBlockchain(&bch)
					if err != nil {
						ui.PrintErrorf("Failed to save wallet: %s", err)
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

			ui.PrintErrorf("Blockchain %s not found", b_name)
		}

	case "remove":
		if b_name == "" {
			ui.PrintErrorf("Usage: blockchain remove [blockchain]")
			return
		}

		for _, b := range w.Blockchains {
			if b.Name == b_name {

				bus.Send("ui", "popup", ui.DlgConfirm(
					"Remove blockchain",
					`
<c>Are you sure you want to remove 
<c>blockchain '`+b_name+"' ?\n",
					func() bool {
						err := w.DeleteBlockchain(b_name)
						if err != nil {
							ui.PrintErrorf("Failed to save wallet: %s", err)
							return false
						}

						ui.Printf("\nBlockchain %s removed\n", b_name)
						return true
					}))
				return
			}
		}

		ui.PrintErrorf("Blockchain %s not found", b_name)

	case "list", "":
		ui.Printf("\nBlockchains:\n")

		sort.Slice(w.Blockchains, func(i, j int) bool {
			return w.Blockchains[i].ChainId < w.Blockchains[j].ChainId
		})

		for _, b := range w.Blockchains {
			ui.Printf("%4d ", b.ChainId)
			ui.Terminal.Screen.AddLink(b.Name, "command b use '"+b.Name+"'", "Use blockchain '"+b.Name+"'", "")
			ui.Printf(" ")
			ui.Terminal.Screen.AddLink(gocui.ICON_EDIT, "command b edit '"+b.Name+"'", "Edit blockchain '"+b.Name+"'", "")
			ui.Terminal.Screen.AddLink(gocui.ICON_DELETE, "command b remove '"+b.Name+"'", "Remove blockchain '"+b.Name+"'", "")
			ui.Printf("\n")
		}

		ui.Printf("\n")
	case "edit":
		if b_name == "" {
			ui.PrintErrorf("Usage: blockchain edit [blockchain]")
			return
		}

		b := w.GetBlockchainByName(b_name)
		if b == nil {
			ui.PrintErrorf("Blockchain %s not found", b_name)
			return
		}

		bus.Send("ui", "popup", ui.DlgBlockchain(b.ChainId))

	case "set":
		if b_name == "" {
			ui.PrintErrorf("Usage: blockchain use [blockchain]")
			return
		}

		if w.GetBlockchainByName(b_name) == nil {
			ui.PrintErrorf("Blockchain %s not found", b_name)
			return
		}

		w.CurrentChain = b_name
		err := w.Save()
		if err != nil {
			ui.PrintErrorf("Failed to save wallet: %s", err)
			return
		}

	default:
		ui.PrintErrorf("Invalid subcommand: %s", subcommand)
	}

}
