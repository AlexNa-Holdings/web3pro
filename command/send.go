package command

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/eth"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/AlexNa-Holdings/web3pro/wallet"
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

	if wallet.CurrentWallet == nil {
		return "", nil, ""
	}

	w := wallet.CurrentWallet

	options := []ui.ACOption{}
	p := cmn.SplitN(input, 6)
	command, bchain, token, from, to, val := p[0], p[1], p[2], p[3], p[4], p[5]

	b := w.GetBlockchain(bchain)

	if val != "" {
		return "", nil, val
	}

	if b != nil && (token == "" || w.GetToken(b.Name, token) == nil) {
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
					Name:   t.Symbol,
					Result: command + " '" + b.Name + "' " + id + " "})
			}
		}
		return "token", &options, token
	}

	if b != nil && w.GetToken(b.Name, token) != nil &&
		from != "" && to != "" && strings.HasSuffix(input, " ") {
		return "", nil, ""
	}

	if b != nil && w.GetToken(b.Name, token) != nil && from != "" && strings.HasSuffix(input, " ") {
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

	if b != nil && w.GetToken(b.Name, token) != nil {
		for _, a := range w.Addresses {
			if cmn.Contains(a.Name+a.Address.String(), from) {
				options = append(options, ui.ACOption{
					Name:   cmn.ShortAddress(a.Address) + " " + a.Name,
					Result: command + " '" + b.Name + "' " + token + " " + a.Address.String() + " "})
			}
		}
		return "from", &options, from
	}

	for _, chain := range w.Blockchains {
		if cmn.Contains(chain.Name, bchain) {
			options = append(options, ui.ACOption{
				Name: chain.Name, Result: command + " '" + chain.Name + "' "})
		}
	}
	return "blockchain", &options, bchain

}

func Send_Process(c *Command, input string) {
	if wallet.CurrentWallet == nil {
		ui.PrintErrorf("\nNo wallet open\n")
		return
	}

	w := wallet.CurrentWallet

	//parse command subcommand parameters
	p := cmn.SplitN(input, 6)
	//execute command
	bchain, token, from, to, amount := p[1], p[2], p[3], p[4], p[5]

	log.Debug().Msgf("Send_Process: %s %s %s %s", bchain, token, to, amount)

	b := w.GetBlockchain(bchain)
	if b == nil {
		ui.PrintErrorf("\nBlockchain not found: %s\n", bchain)
		return
	}

	t := w.GetToken(b.Name, token)
	if t == nil {
		ui.PrintErrorf("\nToken not found: %s\n", token)
		return
	}

	if !common.IsHexAddress(from) {
		ui.PrintErrorf("\nInvalid address from: %s\n", from)
		return
	}

	a_from := w.GetAddress(from)
	if a_from == nil {
		ui.PrintErrorf("\nAddress not found: %s\n", from)
		return
	}

	if to == "" || amount == "" {
		ui.Gui.ShowPopup(ui.DlgSend(b, t, a_from, to, amount))
		return
	}

	if !common.IsHexAddress(from) {
		ui.PrintErrorf("\nInvalid address from: %s\n", from)
		return
	}

	if !common.IsHexAddress(to) {
		ui.PrintErrorf("\nInvalid address to: %s\n", to)
		return
	}

	amt, err := t.Str2Value(amount)
	if err != nil {
		log.Error().Err(err).Msgf("Str2Value(%s) err: %v", amount, err)
		ui.Notification.ShowErrorf("Invalid amount: %s", amount)
		return
	}

	eth.HailToSend(b, t, a_from, common.HexToAddress(to), amt)
}
