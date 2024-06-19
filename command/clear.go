package command

import "github.com/AlexNa-Holdings/web3pro/ui"

func NewClearCommand() *Command {
	return &Command{
		Command:      "clear",
		ShortCommand: "",
		Usage: `
Usage: clear [COMMAND]

This command cleans the terminal screen
		`,
		Help:             `Clean the screen`,
		Process:          Clear_Process,
		AutoCompleteFunc: nil,
	}
}

func Clear_Process(c *Command, input string) {
	ui.Terminal.Screen.Clear()
}
