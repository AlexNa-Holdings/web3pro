package command

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

func NewSendCommand() *Command {
	return &Command{
		Command:      "send",
		ShortCommand: "",
		Usage: `
Usage: send [BLOCKCHAIN] [TOKEN/ADDRESS] [FROM] [TO] amount
`,
		Help:             `Send tokens`,
		Process:          Send_Process,
		AutoCompleteFunc: Send_AutoComplete,
	}
}

func Send_AutoComplete(input string) (string, *[]ui.ACOption, string) {

	if cmn.CurrentWallet == nil {
		return "", nil, ""
	}

	w := cmn.CurrentWallet

	options := []ui.ACOption{}
	p := cmn.SplitN(input, 6)
	command, bchain, token, from, to, val := p[0], p[1], p[2], p[3], p[4], p[5]

	last_param := len(p) - 1
	for last_param > 0 && p[last_param] == "" {
		last_param--
	}

	if strings.HasSuffix(input, " ") {
		last_param++
	}

	b := w.GetBlockchain(bchain)

	var t *cmn.Token
	if b != nil {
		t = w.GetToken(b.Name, token)
	}

	switch last_param {
	case 1:
		for _, chain := range w.Blockchains {
			if cmn.Contains(chain.Name, bchain) {
				options = append(options, ui.ACOption{
					Name: chain.Name, Result: command + " '" + chain.Name + "' "})
			}
		}
		return "blockchain", &options, bchain
	case 2:
		if b != nil {

			for _, t := range w.Tokens {
				if t.Blockchain != b.Name {
					continue
				}
				if cmn.Contains(t.Symbol, token) || cmn.Contains(t.Address.String(), token) || cmn.Contains(t.Name, token) {

					id := t.Symbol
					if !t.Unique {
						id = t.Address.String()
					}

					options = append(options, ui.ACOption{
						Name:   fmt.Sprintf("%-6s %s", t.Symbol, t.GetPrintName()),
						Result: command + " '" + b.Name + "' " + id + " "})
				}
			}
			return "token", &options, token
		}
	case 3:
		if b != nil && t != nil {
			for _, a := range w.Addresses {
				if cmn.Contains(a.Name+a.Address.String(), from) {
					options = append(options, ui.ACOption{
						Name:   cmn.ShortAddress(a.Address) + " " + a.Name,
						Result: command + " '" + b.Name + "' " + token + " " + a.Address.String() + " "})
				}
			}
			return "from", &options, from
		}
	case 4:
		if b != nil && t != nil {
			for _, a := range w.Addresses {
				if cmn.Contains(a.Name+a.Address.String(), to) {
					options = append(options, ui.ACOption{
						Name: cmn.ShortAddress(a.Address) + " " + a.Name,
						Result: command + " '" + b.Name + "' " + token + " " +
							from + " " + a.Address.String() + " "})
				}
			}
			return "to", &options, from
		}
	}

	return "", nil, val
}

func Send_Process(c *Command, input string) {
	if cmn.CurrentWallet == nil {
		ui.PrintErrorf("No wallet open")
		return
	}

	w := cmn.CurrentWallet

	//parse command subcommand parameters
	p := cmn.SplitN(input, 6)
	//execute command
	bchain, token, from, to, amount := p[1], p[2], p[3], p[4], p[5]

	b := w.GetBlockchain(bchain)
	if b == nil {
		ui.PrintErrorf("Blockchain not found: %s", bchain)
		return
	}

	t := w.GetToken(b.Name, token)
	if t == nil {
		ui.PrintErrorf("Token not found: %s", token)
		return
	}

	if !common.IsHexAddress(from) {
		ui.PrintErrorf("Invalid address from: %s", from)
		return
	}

	a_from := w.GetAddress(from)
	if a_from == nil {
		ui.PrintErrorf("Address not found: %s", from)
		return
	}

	if to == "" || amount == "" {
		bus.Send("ui", "popup", ui.DlgSend(b, t, a_from, to, amount))
		return
	}

	if !common.IsHexAddress(from) {
		ui.PrintErrorf("Invalid address from: %s", from)
		return
	}

	if !common.IsHexAddress(to) {
		ui.PrintErrorf("Invalid address to: %s", to)
		return
	}

	amt, err := t.Str2Wei(amount)
	if err != nil || amt.Cmp(big.NewInt(0)) <= 0 {
		log.Error().Err(err).Msgf("Str2Value(%s)", amount)
		ui.Notification.ShowErrorf("Invalid amount: %s", amount)
		return
	}

	bus.Send("eth", "send", &bus.B_EthSend{
		ChainId: b.ChainId,
		Token:   t.Symbol,
		From:    a_from.Address,
		To:      common.HexToAddress(to),
		Amount:  amt,
	})

}
