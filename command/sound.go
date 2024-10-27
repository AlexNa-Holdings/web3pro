package command

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
)

var sound_subcommands = []string{"list", "set", "play", "on", "off"}

func NewSoundCommand() *Command {
	return &Command{
		Command:      "sound",
		ShortCommand: "",
		Usage: `
Usage: 

  sound  - List of sound alerts
  set    - Set sound alert
  play   - Play sound alert
  on     - Turn on sound alert
  off    - Turn off sound alert

		`,
		Help:             `Configure sound`,
		Process:          Sound_Process,
		AutoCompleteFunc: Sound_AutoComplete,
	}
}

func Sound_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}
	p := cmn.Split3(input)
	command, subcommand, _ := p[0], p[1], p[2]

	if !cmn.IsInArray(sound_subcommands, subcommand) {
		for _, sc := range sound_subcommands {
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	}

	return "", &options, ""
}

func Sound_Process(c *Command, input string) {
	p := cmn.Split3(input)
	_, subcommand, param := p[0], p[1], p[2]

	w := cmn.CurrentWallet
	if w == nil {
		ui.PrintErrorf("Wallet not found")
		return
	}

	switch subcommand {
	case "list", "":

		if w.SoundOn {
			ui.Printf("\nSound alert is on\n")
		} else {
			ui.Printf("\nSound alert is off\n")
		}

		resp := bus.Fetch("sound", "list", nil)
		if resp.Error != nil {
			ui.PrintErrorf("Error listing sounds: %v", resp.Error)
			return
		}

		l, ok := resp.Data.([]string)
		if !ok {
			ui.PrintErrorf("Error listing sounds: %v", resp.Error)
			return
		}

		ui.Printf("Current sound alert: %s\n", w.Sound)

		ui.Printf("\nAvailable Alerts:\n")

		n := 1
		for _, s := range l {
			ui.Printf("%02d %-10s ", n, s)

			ui.Terminal.Screen.AddLink(
				cmn.ICON_SEND,
				"command sound play '"+s+"' ",
				"Play",
				"")
			ui.Terminal.Screen.AddLink(
				cmn.ICON_PROMOTE,
				"command sound set '"+s+"' ",
				"Set",
				"")
			ui.Printf("\n")

			n++
		}

		ui.Printf("\n")

		if len(l) == 0 {
			ui.PrintErrorf("No sounds available")
			return
		}

		ui.Printf("\n")

		ui.Flush()
	case "set":
		resp := bus.Fetch("sound", "list", nil)
		if resp.Error != nil {
			ui.PrintErrorf("Error listing sounds: %v", resp.Error)
			return
		}

		l, ok := resp.Data.([]string)
		if !ok {
			ui.PrintErrorf("Error listing sounds: %v", resp.Error)
			return
		}

		for _, s := range l {
			if s == param {
				w.Sound = param
				err := w.Save()
				if err != nil {
					ui.PrintErrorf("Error saving wallet: %v", err)
					return
				}
				ui.Printf("\nSound alert set to: %s\n", param)
				return
			}
		}

		ui.PrintErrorf("Sound alert not found: %s", param)
	case "play":
		if param == "" {
			param = w.Sound
		}
		bus.Send("sound", "play", param)
	case "on":
		w.SoundOn = true
		err := w.Save()
		if err != nil {
			ui.PrintErrorf("Error saving wallet: %v", err)
			return
		}

		ui.Printf("\nSound alert turned on\n")
	case "off":
		w.SoundOn = false
		err := w.Save()
		if err != nil {
			ui.PrintErrorf("Error saving wallet: %v", err)
			return
		}

	default:
		ui.PrintErrorf("Invalid subcommand: %s", subcommand)
	}

}
