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
	Help             string
	Process          CommandProcessFunc
	AutoCompleteFunc AutoCompleteFunc
}

var Commands []*Command

func Init() {
	Commands = []*Command{
		NewHelpCommand(),
		NewWalletCommand(),
		NewThemeCommand(),
		NewClearCommand(),
		NewBlockchainCommand(),
	}
}

func Process(input string) {
	command := strings.Split(input, " ")[0]

	for _, cmd := range Commands {
		if cmd.Command == command || cmd.ShortCommand == command {
			cmd.Process(cmd, input)
			return
		}
	}

	ui.PrintErrorf("\nUnknown command: %s\n\n", command)
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
		if cmd.Command == command || (cmd.ShortCommand != "" && cmd.ShortCommand == command) {
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
	}

	for _, cmd := range Commands {
		if cmd.Command == command || cmd.ShortCommand == command {
			if cmd.AutoCompleteFunc != nil {
				return cmd.AutoCompleteFunc(input)
			}
		}
	}

	return command, &options, ""
}

// removes the first word from the input string
func Params(s string) (string, string) {
	first_word := strings.Split(s, " ")[0]
	return strings.TrimLeft(strings.Replace(s, first_word, "", 1), " \t"), first_word
}
