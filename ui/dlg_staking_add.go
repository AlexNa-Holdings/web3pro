package ui

import (
	"fmt"
	"strconv"

	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ethereum/go-ethereum/common"
)

func DlgStakingAdd(b *cmn.Blockchain) *gocui.Popup {
	template := fmt.Sprintf(`
         Chain: %s
          Name: <input id:name size:32 value:"">
       Contract: <input id:contract size:42 value:"">
            URL: <input id:url size:42 value:"">
   Staked Token: <input id:staked_token size:42 value:"">
   Balance Func: <input id:balance_func size:20 value:"balanceOf">
   Validator ID: <input id:validator_id size:10 value:""> (for native staking)

Reward 1 Token: <input id:reward1_token size:42 value:"">
  Reward 1 Func: <input id:reward1_func size:20 value:"earned">

Reward 2 Token: <input id:reward2_token size:42 value:"">
  Reward 2 Func: <input id:reward2_func size:20 value:"">

<c><button text:Ok tip:"add staking contract">  <button text:Cancel>`, b.Name)

	return &gocui.Popup{
		Title: "Add Staking",
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
					name := v.GetInput("name")

					if name == "" {
						Notification.ShowError("Name cannot be empty")
						break
					}

					contractAddr := v.GetInput("contract")
					c := common.HexToAddress(contractAddr)

					if c == (common.Address{}) {
						Notification.ShowError("Invalid contract address")
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

					// Reward 1 (required)
					reward1TokenAddr := v.GetInput("reward1_token")
					reward1Token := common.HexToAddress(reward1TokenAddr)
					if reward1Token == (common.Address{}) {
						// Default to staked token
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

					w := cmn.CurrentWallet
					if w == nil {
						Notification.ShowError("No wallet open")
						break
					}

					existing := w.GetStaking(b.ChainId, c)
					if existing != nil {
						Notification.ShowError("Staking contract already exists")
						break
					}

					err := w.AddStaking(&cmn.Staking{
						Name:         name,
						ChainId:      b.ChainId,
						Contract:     c,
						URL:          url,
						StakedToken:  st,
						BalanceFunc:  balanceFunc,
						Reward1Token: reward1Token,
						Reward1Func:  reward1Func,
						Reward2Token: reward2Token,
						Reward2Func:  reward2Func,
						ValidatorId:  validatorId,
					})

					if err != nil {
						Notification.ShowErrorf("Error adding staking: %s", err)
						break
					}
					Notification.Showf("Staking %s created", name)
					Gui.HidePopup()

				case "button Cancel":
					Gui.HidePopup()
				}
			}
		},
		Template: template,
	}
}
