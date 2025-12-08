package ui

import (
	"io"
	"os"
	"regexp"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
)

// extractWalletName extracts wallet name from backup filename
// Format: walletName_YYYY-MM-DD_HH-MM-SS
func extractWalletName(backupFile string) string {
	// Match the timestamp suffix pattern: _YYYY-MM-DD_HH-MM-SS
	re := regexp.MustCompile(`_\d{4}-\d{2}-\d{2}_\d{2}-\d{2}-\d{2}$`)
	return re.ReplaceAllString(backupFile, "")
}

func DlgWalletRestore(backupFile string) *gocui.Popup {
	backupPath := cmn.DataFolder + "/wallets/backups/" + backupFile
	walletName := extractWalletName(backupFile)

	return &gocui.Popup{
		Title: "Restore Wallet from " + backupFile,
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {

			if hs != nil {
				switch hs.Value {
				case "button Ok":
					pass := v.GetInput("pass")

					// Try to open the backup file to verify password
					_, err := cmn.OpenFromFile(backupPath, pass)
					if err != nil {
						Notification.ShowErrorf("Error opening backup: %s", err)
						v.SetInput("pass", "")
						v.SetFocus(0)
						break
					}

					targetPath := cmn.DataFolder + "/wallets/" + walletName

					// Check if target wallet exists
					_, err = os.Stat(targetPath)
					walletExists := err == nil

					Gui.HidePopup()

					// Show confirmation dialog
					var confirmMsg string
					if walletExists {
						confirmMsg = "Are you sure you want to overwrite wallet '" + walletName + "'?"
					} else {
						confirmMsg = "Restore wallet '" + walletName + "'?"
					}

					bus.Send("ui", "popup", DlgConfirm("Confirm Restore", confirmMsg, func() bool {
						// Copy backup to wallet location
						srcFile, err := os.Open(backupPath)
						if err != nil {
							Notification.ShowErrorf("Failed to open backup: %s", err)
							return false
						}
						defer srcFile.Close()

						dstFile, err := os.Create(targetPath)
						if err != nil {
							Notification.ShowErrorf("Failed to create wallet file: %s", err)
							return false
						}
						defer dstFile.Close()

						_, err = io.Copy(dstFile, srcFile)
						if err != nil {
							Notification.ShowErrorf("Failed to restore wallet: %s", err)
							return false
						}

						Notification.Showf("Wallet '%s' restored successfully", walletName)
						return true
					}))

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: `
 Password: <input id:pass masked:true size:24>
 <c>
 <button text:Ok tip:"verify and restore">  <button text:Cancel>`,
	}
}
