package command

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/ui"
)

type CommandProcessFunc func(*Command, string)
type AutoCompleteFunc func(string) (string, *[]ui.ACOption, string)

type Command struct {
	Command          string
	ShortCommand     string
	Usage            string
	Process          CommandProcessFunc
	AutoCompleteFunc AutoCompleteFunc
}

var Commands = []*Command{
	NewThemeCommand(),
}

func Process(input string) {
	command := strings.Split(input, " ")[0]

	for _, cmd := range Commands {
		if cmd.Command == command || cmd.ShortCommand == command {
			cmd.Process(cmd, input)
			return
		}
	}

	ui.PrintErrorf("Unknown command: %s\n", command)
}

func AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}

	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		return "", &options, ""
	}

	command := tokens[0]

	is_command := false
	for _, cmd := range Commands {
		if cmd.Command == command || cmd.ShortCommand == command {
			is_command = true
			break
		}
	}

	if len(tokens) == 1 && !is_command {
		for _, cmd := range Commands {
			if strings.Contains(cmd.Command, command) || strings.Contains(cmd.ShortCommand, command) {
				options = append(options, ui.ACOption{Name: cmd.Command, Result: cmd.Command + " "})
			}
		}
		return "command", &options, command
	} else {
		for _, cmd := range Commands {
			if cmd.Command == command || cmd.ShortCommand == command {
				if cmd.AutoCompleteFunc != nil {
					return cmd.AutoCompleteFunc(input)
				}
			}
		}
	}

	return command, &options, ""
}
