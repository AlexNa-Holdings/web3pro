package command

import (
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/ui"
)

func NewThemeCommand() *Command {
	return &Command{
		Command:      "theme",
		ShortCommand: "",
		Help:         "change/show the UI theme",
		Usage:        "theme <theme>",
		Process:      CommandProcessFunc(processTheme),
	}
}

func processTheme(cmd *Command, input string) {
	//parse command subcommand parameters
	tokens := strings.Split(input, " ")
	if len(tokens) < 2 {
		fmt.Fprintln(ui.Terminal.Screen, cmd.Usage)
		return
	}
	//execute command
	subcommand := tokens[1]

	switch subcommand {
	case "list":
		ui.Printf("\nAvailable themes:\n")

		for _, theme := range ui.Themes {
			ui.Printf(theme.Name + "\n")
		}
	case "demo":
		if len(tokens) < 3 || tokens[2] == "" {
			demoTheme(ui.CurrentTheme.Name)
		} else {
			demoTheme(tokens[2])
		}
	default:
		ui.PrintErrorf("Unknown subcommand: %s\n", subcommand)
	}
}

func DL(s string) string {
	const L = 45

	// add spacec if neede to make it L characters
	if len(s) < L {
		s = s + strings.Repeat(" ", L-len(s))
	}

	return s
}

func demoTheme(theme string) {
	t, ok := ui.Themes[theme]
	if !ok {
		ui.PrintErrorf("Unknown theme: %s\n", theme)
		return
	}

	ui.Printf("\nDemo theme: %s\n", theme)

	ui.Printf(ui.FB(t.FgColor, t.BgColor) + DL("FgColor / BgColor") + "\n")
	ui.Printf(ui.FB(t.SelFgColor, t.SelBgColor) + DL("SelFgColor / SelBgColor") + "\n")
	ui.Printf(ui.FB(t.ActionFgColor, t.ActionBgColor) + DL("ActionFgColor / ActionBgColor") + "\n")
	ui.Printf(ui.FB(t.ActionSelFgColor, t.ActionSelBgColor) + DL("ActionSelFgColor / ActionSelBgColor") + "\n")
	ui.Printf(ui.FB(t.ErrorFgColor, t.BgColor) + DL("ErrorFgColor / BgColor") + "\n")
	ui.Printf(ui.FB(t.FrameColor, t.BgColor) + DL("FrameColor / BgColor") + "\n\n")

	ui.ResetColors()

}
