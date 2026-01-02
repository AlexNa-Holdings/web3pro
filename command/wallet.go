package command

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/ui"
)

var wallet_subcommands = []string{"backup", "close", "create", "list", "password", "restore", "open"}

func NewWalletCommand() *Command {
	return &Command{
		Command:      "wallet",
		ShortCommand: "w",
		Subcommands:  wallet_subcommands,
		Usage: `
Usage: wallet [COMMAND]

Manage wallets

Commands:
  open <wallet>    Open wallet
  create           Create new wallet
  close            Close current wallet
  list             List wallets
  backup           Backup current wallet
  restore <backup> Restore wallet from backup
  password         Change wallet password

		`,
		Help:             `Manage wallets`,
		Process:          Wallet_Process,
		AutoCompleteFunc: Wallet_AutoComplete,
	}
}

func Wallet_AutoComplete(input string) (string, *[]ui.ACOption, string) {
	options := []ui.ACOption{}
	p := cmn.Split3(input)
	command, subcommand, param := p[0], p[1], p[2]

	if !cmn.IsInArray(wallet_subcommands, subcommand) {
		for _, sc := range wallet_subcommands {
			// Only show restore when no wallet is open
			if sc == "restore" && cmn.CurrentWallet != nil {
				continue
			}
			if input == "" || strings.Contains(sc, subcommand) {
				options = append(options, ui.ACOption{Name: sc, Result: command + " " + sc + " "})
			}
		}
		return "action", &options, subcommand
	}

	if subcommand == "open" {

		if param != "" && !strings.HasSuffix(param, " ") {
			return "", nil, param
		}

		files := cmn.WalletList()

		for _, file := range files {
			if param == "" || strings.Contains(file, param) {
				options = append(options, ui.ACOption{Name: file, Result: command + " open " + file + " "})
			}
		}

		return "file", &options, param
	}

	if subcommand == "restore" {

		if param != "" && !strings.HasSuffix(param, " ") {
			return "", nil, param
		}

		files := cmn.BackupList()

		for _, file := range files {
			if param == "" || strings.Contains(file, param) {
				options = append(options, ui.ACOption{Name: file, Result: command + " restore " + file + " "})
			}
		}

		return "backup", &options, param
	}

	return "", &options, ""
}

func Wallet_Process(c *Command, input string) {
	//parse command subcommand parameters
	tokens := cmn.Split3(input)
	//execute command
	subcommand := tokens[1]

	switch subcommand {
	case "open":
		if len(tokens) != 3 {
			ui.PrintErrorf("Please specify wallet name")
			return
		}
		bus.Send("ui", "popup", ui.DlgWaletOpen(tokens[2]))

	case "create":
		bus.Send("ui", "popup", ui.DlgWaletCreate())
	case "close":
		if cmn.CurrentWallet != nil {
			cmn.CurrentWallet = nil
			ui.Terminal.SetCommandPrefix(ui.DEFAULT_COMMAND_PREFIX)
			ui.Notification.Show("Wallet closed")
		} else {
			ui.PrintErrorf("No wallet open")
		}
	case "list", "":
		files := cmn.WalletList()
		if files == nil {
			ui.PrintErrorf("Error reading directory")
			return
		}

		ui.Printf("\nWallets:\n")

		for _, file := range files {
			ui.Terminal.Screen.AddLink(file, "command w open "+file, "Open wallet "+file, "")
			ui.Printf("\n")
		}

		ui.Printf("\n")

	case "backup":
		if cmn.CurrentWallet == nil {
			ui.PrintErrorf("No wallet open")
			return
		}

		walletPath := cmn.CurrentWallet.GetFilePath()
		walletName := filepath.Base(walletPath)
		backupDir := cmn.DataFolder + "/wallets/backups"

		// Create backup directory if it doesn't exist
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			ui.PrintErrorf("Failed to create backup directory: %s", err)
			return
		}

		// Generate backup filename with timestamp
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		backupPath := fmt.Sprintf("%s/%s_%s", backupDir, walletName, timestamp)

		// Copy the wallet file
		srcFile, err := os.Open(walletPath)
		if err != nil {
			ui.PrintErrorf("Failed to open wallet file: %s", err)
			return
		}
		defer srcFile.Close()

		dstFile, err := os.Create(backupPath)
		if err != nil {
			ui.PrintErrorf("Failed to create backup file: %s", err)
			return
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			ui.PrintErrorf("Failed to copy wallet file: %s", err)
			return
		}

		ui.Printf("Wallet backed up to: %s\n", backupPath)

	case "restore":
		if cmn.CurrentWallet != nil {
			ui.PrintErrorf("Please close the current wallet first")
			return
		}

		if len(tokens) != 3 || tokens[2] == "" {
			ui.PrintErrorf("Please specify backup file name")
			return
		}

		backupFile := tokens[2]
		bus.Send("ui", "popup", ui.DlgWalletRestore(backupFile))

	case "password":
		if cmn.CurrentWallet == nil {
			ui.PrintErrorf("No wallet open")
			return
		}
		bus.Send("ui", "popup", ui.DlgWalletPassword())

	default:
		ui.PrintErrorf("Invalid subcommand: %s", subcommand)
	}

}
