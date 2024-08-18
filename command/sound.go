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
	p := cmn.Split(input)
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
	p := cmn.Split(input)
	_, subcommand, param := p[0], p[1], p[2]

	w := cmn.CurrentWallet
	if w == nil {
		ui.PrintErrorf("\nWallet not found\n")
		return
	}

	switch subcommand {
	case "list", "":
		ui.Printf("\nAvailable Alerts:\n")

		resp := bus.Fetch("sound", "list", nil)
		if resp.Error != nil {
			ui.PrintErrorf("\nError listing sounds: %v\n", resp.Error)
			return
		}

		l, ok := resp.Data.([]string)
		if !ok {
			ui.PrintErrorf("\nError listing sounds: %v\n", resp.Error)
			return
		}

		n := 1
		for _, s := range l {
			ui.Printf("%02d %s\n", n, s)
			n++
		}

		ui.Printf("\n")

		if len(l) == 0 {
			ui.PrintErrorf("\nNo sounds available\n")
			return
		}

		ui.Printf("\n")

		ui.Flush()
	case "set":
		resp := bus.Fetch("sound", "list", nil)
		if resp.Error != nil {
			ui.PrintErrorf("\nError listing sounds: %v\n", resp.Error)
			return
		}

		l, ok := resp.Data.([]string)
		if !ok {
			ui.PrintErrorf("\nError listing sounds: %v\n", resp.Error)
			return
		}

		for _, s := range l {
			if s == param {
				w.Sound = param
				err := w.Save()
				if err != nil {
					ui.PrintErrorf("\nError saving wallet: %v\n", err)
					return
				}
				ui.Printf("\nSound alert set to: %s\n", param)
				return
			}
		}

		ui.PrintErrorf("\nSound alert not found: %s\n", param)
	case "play":
		if param == "" {
			param = w.Sound
		}
		bus.Fetch("sound", "play", param)
	case "on":
		w.SoundOn = true
		err := w.Save()
		if err != nil {
			ui.PrintErrorf("\nError saving wallet: %v\n", err)
			return
		}

		ui.Printf("\nSound alert turned on\n")
	case "off":
		w.SoundOn = false
		err := w.Save()
		if err != nil {
			ui.PrintErrorf("\nError saving wallet: %v\n", err)
			return
		}

	default:
		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
	}

}
