package command

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
)

var websocket_subcommands = []string{"list"}

func NewWebSocketCommand() *Command {
	return &Command{
		Command:      "websocket",
		ShortCommand: "ws",
		Usage: `
Usage: 

  websocket(ws)  - List of active browser connections

		`,
		Help:             `List browse connections`,
		Process:          Ws_Process,
		AutoCompleteFunc: Ws_AutoComplete,
	}
}

func Ws_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}
	p := cmn.Split(input)
	command, subcommand, _ := p[0], p[1], p[2]

	if !cmn.IsInArray(websocket_subcommands, subcommand) {
		for _, sc := range websocket_subcommands {
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	}

	return "", &options, ""
}

func Ws_Process(c *Command, input string) {
	if cmn.CurrentWallet == nil {
		ui.PrintErrorf("\nNo wallet open\n")
		return
	}

	p := cmn.Split(input)
	_, subcommand := p[0], p[1]

	switch subcommand {
	case "list", "":
		ui.Printf("\nActive browser connections:\n")

		resp := bus.Fetch("ws", "list", nil)
		if resp.Error != nil {
			ui.PrintErrorf("\nError listing connections: %v\n", resp.Error)
			return
		}

		l, ok := resp.Data.(bus.B_WsList_Response)
		if !ok {
			ui.PrintErrorf("\nError listing: %v\n", resp.Error)
			return
		}

		n := 1
		for _, c := range l {
			ui.Printf("%02d %s\n", n, c.Agent)
			n++
		}

		ui.Printf("\n")

		if len(l) == 0 {
			ui.PrintErrorf("\nNo connections\n")
			return
		}

		ui.Printf("\n")

		ui.Flush()

	default:
		ui.PrintErrorf("\nInvalid subcommand: %s\n", subcommand)
	}
}
