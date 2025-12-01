package ui

import (
	"fmt"
	"strconv"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
)

func DlgStakingEdit(b *cmn.Blockchain, s *cmn.Staking) *gocui.Popup {
	template := fmt.Sprintf(`
         Chain: %s
      Contract: %s
          Name: <input id:name size:32 value:"">
           URL: <input id:url size:42 value:"">
  Staked Token: <input id:staked_token size:42 value:"">
  Balance Func: <input id:balance_func size:20 value:"">
  Validator ID: <input id:validator_id size:10 value:""> (for native staking)

Reward 1 Token: <input id:reward1_token size:42 value:"">
  Reward 1 Func: <input id:reward1_func size:20 value:"">

Reward 2 Token: <input id:reward2_token size:42 value:"">
  Reward 2 Func: <input id:reward2_func size:20 value:"">

<c><button text:Ok tip:"save changes">  <button text:Cancel>`, b.Name, s.Contract.Hex())

	return &gocui.Popup{
		Title: "Edit Staking",
		OnOverHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				Bottom.Printf(hs.Tip)
			} else {
				Bottom.Printf("")
			}
		},
		OnOpen: func(v *gocui.View) {
			v.SetInput("name", s.Name)
			v.SetInput("url", s.URL)
			v.SetInput("staked_token", s.StakedToken.Hex())
			v.SetInput("balance_func", s.BalanceFunc)
			if s.ValidatorId > 0 {
				v.SetInput("validator_id", strconv.FormatUint(s.ValidatorId, 10))
			}
			v.SetInput("reward1_token", s.Reward1Token.Hex())
			v.SetInput("reward1_func", s.Reward1Func)
			v.SetInput("reward2_token", s.Reward2Token.Hex())
			v.SetInput("reward2_func", s.Reward2Func)
		},
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {

			if hs != nil {
				switch hs.Value {
				case "button Ok":
					name := v.GetInput("name")

					if name == "" {
						Notification.ShowError("Name cannot be empty")
						break
					}

					w := cmn.CurrentWallet
					if w == nil {
						Notification.ShowError("No wallet open")
						break
					}

					existing := w.GetStaking(b.ChainId, s.Contract)
					if existing == nil {
						Notification.ShowError("Staking not found")
						break
					}

					stakedTokenAddr := v.GetInput("staked_token")
					st := common.HexToAddress(stakedTokenAddr)

					if st == (common.Address{}) {
						Notification.ShowError("Invalid staked token address")
						break
					}

					balanceFunc := v.GetInput("balance_func")
					if balanceFunc == "" {
						balanceFunc = "balanceOf"
					}

					url := v.GetInput("url")

					// Reward 1
					reward1TokenAddr := v.GetInput("reward1_token")
					reward1Token := common.HexToAddress(reward1TokenAddr)
					if reward1Token == (common.Address{}) {
						reward1Token = st
					}
					reward1Func := v.GetInput("reward1_func")
					if reward1Func == "" {
						reward1Func = "earned"
					}

					// Reward 2 (optional)
					reward2TokenAddr := v.GetInput("reward2_token")
					reward2Token := common.HexToAddress(reward2TokenAddr)
					reward2Func := v.GetInput("reward2_func")

					// Validator ID (optional, for native staking like Monad)
					var validatorId uint64
					validatorIdStr := v.GetInput("validator_id")
					if validatorIdStr != "" {
						vid, err := strconv.ParseUint(validatorIdStr, 10, 64)
						if err != nil {
							Notification.ShowError("Invalid validator ID")
							break
						}
						validatorId = vid
					}

					// Update the staking
					existing.Name = name
					existing.URL = url
					existing.StakedToken = st
					existing.BalanceFunc = balanceFunc
					existing.ValidatorId = validatorId
					existing.Reward1Token = reward1Token
					existing.Reward1Func = reward1Func
					existing.Reward2Token = reward2Token
					existing.Reward2Func = reward2Func

					err := cmn.CurrentWallet.Save()
					if err != nil {
						Notification.ShowErrorf("Error updating staking: %s", err)
						break
					}
					Notification.Showf("Staking %s changed", name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: template,
	}
}
