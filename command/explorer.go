package command

import (
	"os"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/ethereum/go-ethereum/common"
)

var explorer_subcommands = []string{"download", "code"}

func NewExplorerCommand() *Command {
	return &Command{
		Command:      "explorer",
		ShortCommand: "x",
		Usage: `
Usage: explorer [command] [params]


Commands:

  download [BLOCKCHAIN] [CONTRACT] - download ABI and code for contract
  code [CONTRACT]                  - show code for contract`,
		Help:             `Explorer API`,
		Process:          Explorer_Process,
		AutoCompleteFunc: Explorer_AutoComplete,
	}
}

func Explorer_AutoComplete(input string) (string, *[]ui.ACOption, string) {

	if cmn.CurrentWallet == nil {
		return "", nil, ""
	}

	w := cmn.CurrentWallet
	if w == nil {
		return "", nil, ""
	}

	options := []ui.ACOption{}
	p := cmn.SplitN(input, 6)
	command, subcommand, bchain := p[0], p[1], p[2]

	last_param := len(p) - 1
	for last_param > 0 && p[last_param] == "" {
		last_param--
	}

	if strings.HasSuffix(input, " ") {
		last_param++
	}

	switch last_param {
	case 0, 1:
		for _, sc := range explorer_subcommands {
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	case 2:
		if subcommand == "download" {
			for _, chain := range w.Blockchains {
				if cmn.Contains(chain.Name, bchain) {
					options = append(options, ui.ACOption{
						Name: chain.Name, Result: command + " " + subcommand + " '" + chain.Name + "' "})
				}
			}
			return "blockchain", &options, bchain
		}
		if subcommand == "code" {
			// check if folder exists
			info, err := os.Stat(cmn.DataFolder + "/contracts")
			if err == nil && info.IsDir() {
				files, err := os.ReadDir(cmn.DataFolder + "/contracts")
				if err == nil {
					for _, f := range files {
						if f.IsDir() {
							options = append(options, ui.ACOption{
								Name: f.Name(), Result: command + " " + subcommand + " '" + f.Name() + "' "})
						}
					}
				}
			}
		}
		return "contract", &options, ""
	}

	return "", nil, ""
}

func Explorer_Process(c *Command, input string) {

	if cmn.CurrentWallet == nil {
		ui.PrintErrorf("No wallet open")
		return
	}

	w := cmn.CurrentWallet
	if w == nil {
		ui.PrintErrorf("Explorer_Process: no wallet")
		return
	}

	p := cmn.SplitN(input, 6)
	_, subcommand, bchain, address := p[0], p[1], p[2], p[3]

	a := common.HexToAddress(address)

	switch subcommand {
	case "download":
		b := w.GetBlockchain(bchain)
		if b == nil {
			ui.PrintErrorf("Explorer_Process: blockchain not found: %v", bchain)
			return
		}

		if b.ExplorerUrl == "" {
			ui.PrintErrorf("Explorer_Process: blockchain %s has no explorer", b.Name)
			return
		}

		// check the address format
		if !common.IsHexAddress(address) {
			ui.PrintErrorf("Explorer_Process: invalid address: %s", address)
			return
		}

		resp := bus.Fetch("explorer", "download-contract", &bus.B_ExplorerDownloadContract{
			Blockchain: b.Name,
			Address:    a,
		})
		if resp.Error != nil {
			ui.PrintErrorf("Error downloading contract: %v", resp.Error)
		} else {
			ui.Printf("Contract downloaded")
		}

	case "code":
		if cmn.Config.Editor == "" {
			ui.PrintErrorf("No editor configured")
			return
		}

		to := common.HexToAddress(p[2])
		if (to == common.Address{}) {
			ui.PrintErrorf("Invalid address")
			return
		}

		// check if folder exists
		info, err := os.Stat(cmn.DataFolder + "/contracts/" + to.String())
		if err != nil || !info.IsDir() {
			ui.PrintErrorf("No contracts folder found")
			return
		}

		cmn.SystemCommand(cmn.Config.Editor + "\"" + cmn.DataFolder + "/contracts/" + to.String() + "\"")

	}
}
