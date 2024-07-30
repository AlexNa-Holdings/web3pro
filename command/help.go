package command

import (
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/rs/zerolog/log"
)

func NewHelpCommand() *Command {
	return &Command{
		Command:      "help",
		ShortCommand: "h",
		Usage: `
Usage: help [COMMAND]

This command shows help information for a specific command.

EXAMPLES:
		help theme
		
		`,
		Help:             `Show help information for a specific command`,
		Process:          Help_Process,
		AutoCompleteFunc: Help_AutoComplete,
	}
}

func Help_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}
	p := cmn.Split(input)
	command, subcommand, _ := p[0], p[1], p[2]

	is_command := false
	for _, cmd := range Commands {
		if cmd.Command == subcommand || cmd.ShortCommand == subcommand {
			is_command = true
			break
		}
	}

	if !is_command {
		for _, sc := range Commands {
			if cmn.Contains(sc.Command, subcommand) || cmn.Contains(sc.ShortCommand, subcommand) {
				options = append(options, ui.ACOption{Name: sc.Command, Result: command + " " + sc.Command})
			}
		}
		return "command", &options, subcommand
	}

	return "", &[]ui.ACOption{}, ""
}

func Help_Process(cmd *Command, input string) {

	test()
	return

	//parse command subcommand parameters
	tokens := strings.Fields(input)
	if len(tokens) < 2 {
		ui.Printf("\nAvailable commands:\n\n")
		for _, sc := range Commands {
			short := ""
			if sc.ShortCommand != "" {
				short = "(" + sc.ShortCommand + ")"
			}
			ui.Printf("%-13s - %s\n", sc.Command+short, sc.Help)
		}

		ui.Printf("\n")
		return
	}
	command := tokens[1]

	for _, sc := range Commands {
		if sc.Command == command || (sc.ShortCommand != "" && sc.ShortCommand == command) {
			fmt.Fprintln(ui.Terminal.Screen, sc.Usage)
			return
		}
	}
}

func test() {

	log.Debug().Msg("test start--------------------------")

	bus.Fetch("ui", "hail", &bus.B_Hail{
		Title: "TEST HAIL",
		Template: `<c><w>
Test

<button text:OK> <button text:Cancel>`,
	})

	log.Debug().Msg("test done")

}
