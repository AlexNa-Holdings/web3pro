package command

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/ui"
)

var theme_subcommands = []string{"list", "demo"}

func NewThemeCommand() *Command {
	return &Command{
		Command:      "theme",
		ShortCommand: "",
		Usage: `
Usage: theme [COMMAND]

This command allows you to change or show the UI theme.

COMMANDS:
		demo [THEME] - show theme colors (default: current theme)
		`,
		Help:             `UI themes management`,
		Process:          Theme_Process,
		AutoCompleteFunc: Theme_AutoComplete,
	}
}

func Theme_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	params, first_word := Params(input)
	re_subcommand := regexp.MustCompile(`^(\w*)$`)

	if m := re_subcommand.FindStringSubmatch(params); m != nil {
		si := m[1]

		options := []ui.ACOption{}

		is_subcommand := false
		for _, sc := range theme_subcommands {
			if sc == si {
				is_subcommand = true
				break
			}
		}

		if !is_subcommand {
			for _, sc := range []string{"list", "demo"} {
				if input == "" || strings.Contains(sc, si) {
					options = append(options, ui.ACOption{Name: sc, Result: first_word + " " + sc})
				}
			}
		}

		return "action", &options, input
	}

	return input, &[]ui.ACOption{}, ""
}

func Theme_Process(cmd *Command, input string) {
	//parse command subcommand parameters
	tokens := strings.Fields(input)
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

		ui.Printf("\n")
	case "demo":
		if len(tokens) < 3 {
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
	ui.Printf(ui.FB(t.ErrorFgColor, t.BgColor) + DL("ErrorFgColor") + "\n")
	ui.Printf(ui.FB(t.EmFgColor, t.BgColor) + DL("EmFgColor") + "\n")
	ui.Printf(ui.FB(t.FrameColor, t.BgColor) + DL("FrameColor / BgColor") + "\n")
	ui.Printf(ui.FB(t.HelpFgColor, t.HelpBgColor) + DL("HelpFgColor / HelpBgColor") + "\n\n")

	ui.ResetColors()

}
