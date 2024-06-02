package command

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/ui"
)

type CommandProcessFunc func(*Command, string)

type Command struct {
	Command      string
	ShortCommand string
	Usage        string
	Process      CommandProcessFunc
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
