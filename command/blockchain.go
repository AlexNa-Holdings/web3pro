package command

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/blockchain"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
)

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
	params, first_word := Params(input)

	re_subcommand := regexp.MustCompile(`^(\w*)$`)
	if m := re_subcommand.FindStringSubmatch(params); m != nil {
		si := m[1]

		is_subcommand := false
		for _, sc := range theme_subcommands {
			if sc == si {
				is_subcommand = true
				break
			}
		}

		if !is_subcommand {
			for _, sc := range []string{"remove", "add", "list", "use"} {
				if input == "" || strings.Contains(sc, si) {
					options = append(options, ui.ACOption{Name: sc, Result: first_word + " " + sc + " "})
				}
			}
		}

		return "action", &options, si
	}

	re_demo := regexp.MustCompile(`^use\s+(\w*)$`)
	if m := re_demo.FindStringSubmatch(params); m != nil {
		t := m[1]

		if wallet.CurrentWallet != nil {
			for _, chain := range wallet.CurrentWallet.Blockchains {
				if t == "" || strings.Contains(chain.Name, t) {
					options = append(options, ui.ACOption{Name: chain.Name, Result: first_word + " use " + chain.Name})
				}
			}
		}

		return "blockchain", &options, t
	}

	re_demo = regexp.MustCompile(`^add\s+(\w*)$`)
	if m := re_demo.FindStringSubmatch(params); m != nil {
		t := m[1]

		for _, chain := range blockchain.PrefefinedBlockchains {
			if t == "" || cmn.Contains(chain.Name, t) {
				options = append(options, ui.ACOption{Name: chain.Name, Result: first_word + " add '" + chain.Name + "' "})
			}
		}

		if t == "" || cmn.Contains("(custom)", t) {
			options = append(options, ui.ACOption{Name: "(custom)", Result: first_word + " add custom "})
		}

		return "blockchain", &options, t
	}

	return input, &options, ""
}

func Blockchain_Process(c *Command, input string) {
	//parse command subcommand parameters
	tokens := strings.Fields(input)
	if len(tokens) < 2 {
		fmt.Fprintln(ui.Terminal.Screen, c.Usage)
		return
	}
	//execute command
	subcommand := tokens[1]

	switch subcommand {
	case "list":
		if wallet.CurrentWallet == nil {
			ui.PrintErrorf("\nNo wallet open\n")
			return
		}

		ui.Printf("\nBlockchains:\n")

		for _, b := range wallet.CurrentWallet.Blockchains {
			ui.Terminal.Screen.AddLink(b.Name, "command b use "+b.Name, "Use blockchain "+b.Name)
			ui.Printf("\n")
		}

		ui.Printf("\n")
	default:
		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
	}

}
