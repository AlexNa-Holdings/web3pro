package command

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/rs/zerolog/log"
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
		NewSignerCommand(),
		NewUsbCommand(),
		NewAddressCommand(),
		NewTokenCommand(),
		NewSendCommand(),
		NewPriceCommand(),
		NewWebSocketCommand(),
		NewAppCommand(),
		NewSoundCommand(),
	}

	go func() {
		ch := bus.Subscribe("ui")
		defer bus.Unsubscribe(ch)
		for {

			msg := <-ch
			if msg.Type == "command" {
				switch msg.Data.(type) {
				case string:
					input := strings.TrimSpace(msg.Data.(string))
					command := strings.Split(input, " ")[0]
					log.Trace().Msgf("Processing command: %s", command)

					found := false
					for _, cmd := range Commands {
						if cmd.Command == command || cmd.ShortCommand == command {
							found = true
							cmd.Process(cmd, input)
							ui.Flush()
							ui.Gui.SetCurrentView("terminal.input")
						}
					}

					if !found {
						ui.PrintErrorf("Unknown command: %s", command)
					}
				}
			}
		}
	}()
}

func AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}
	p := cmn.Split(input)
	command, _, _ := p[0], p[1], p[2]

	if command == "" {
		return "", &options, ""
	}

	for _, cmd := range Commands {
		if cmd.Command == command || cmd.ShortCommand == command {
			if cmd.AutoCompleteFunc != nil {
				return cmd.AutoCompleteFunc(input)
			}
		}
	}

	for _, cmd := range Commands {
		if cmn.Contains(cmd.Command, command) || cmn.Contains(cmd.ShortCommand, command) {

			text := cmd.Command
			if cmd.ShortCommand != "" {
				text += " (" + cmd.ShortCommand + ")"
			}

			options = append(options, ui.ACOption{Name: text, Result: cmd.Command + " "})
		}
	}

	return "command", &options, command
}
