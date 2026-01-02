package ui

import (
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
)

func DlgWalletPassword() *gocui.Popup {
	return &gocui.Popup{
		Title: "Change Password",
		OnOverHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				Bottom.Printf(hs.Tip)
			} else {
				Bottom.Printf("")
			}
		},
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {

			if hs != nil {
				switch hs.Value {
				case "button Ok":
					oldPass := v.GetInput("old_pass")
					newPass := v.GetInput("new_pass")
					retype := v.GetInput("retype")

					if !cmn.CurrentWallet.VerifyPassword(oldPass) {
						Notification.ShowError("Old password is incorrect")
						v.SetInput("old_pass", "")
						v.SetFocus(0)
						break
					}

					if newPass != retype {
						Notification.ShowError("New passwords do not match")
						v.SetInput("new_pass", "")
						v.SetInput("retype", "")
						v.SetFocus(1)
						break
					}

					err := cmn.CurrentWallet.ChangePassword(newPass)
					if err != nil {
						Notification.ShowErrorf("Error changing password: %s", err)
						break
					}

					Notification.Show("Password changed successfully")
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: `
 Old Password: <input id:old_pass masked:true size:24>
 New Password: <input id:new_pass masked:true size:24>
     (Retype): <input id:retype masked:true size:24>
<c>
<button text:Ok tip:"change password">  <button text:Cancel>`,
	}
}
