package command

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
)

var app_subcommands = []string{"remove", "list"}

func NewAppCommand() *Command {
	return &Command{
		Command:      "application",
		ShortCommand: "app",
		Usage: `
Usage: application [COMMAND]

Manage web applications (origins)

Commands:
  list         - List web applications
  remove [URL] - Remove address  
		`,
		Help:             `Manage connected web applications`,
		Process:          App_Process,
		AutoCompleteFunc: App_AutoComplete,
	}
}

func App_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}

	p := cmn.Split(input)
	command, subcommand, param := p[0], p[1], p[2]

	if !cmn.IsInArray(app_subcommands, subcommand) {
		for _, sc := range app_subcommands {
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	}

	if subcommand == "remove" {
		for _, o := range cmn.CurrentWallet.Origins {
			if cmn.Contains(o.URL, param) {
				options = append(options, ui.ACOption{
					Name:   o.URL,
					Result: command + " " + subcommand + " '" + o.URL + "'"})
			}
		}
		return "application", &options, param
	}

	return "", &options, ""
}

func App_Process(c *Command, input string) {
	if cmn.CurrentWallet == nil {
		ui.PrintErrorf("\nNo wallet open\n")
		return
	}

	w := cmn.CurrentWallet

	p := cmn.Split(input)
	_, subcommand := p[0], p[1]

	switch subcommand {
	case "list", "":
		ui.Printf("\nConnected web applications:\n")

		for _, o := range w.Origins {
			ui.Printf("%s\n", o.URL)
		}
	case "remove":
		if len(p) < 3 {
			ui.PrintErrorf("Missing URL\n")
			return
		}

		url := p[2]

		w.RemoveOrigin(url)
		err := w.Save()
		if err != nil {
			ui.PrintErrorf("Error saving wallet: %v\n", err)
			return
		}

		ui.Printf("Removed %s\n", url)
	}
}
