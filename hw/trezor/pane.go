package trezor

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/AlexNa-Holdings/web3pro/ui"
	"github.com/rs/zerolog/log"
)

var TP_Id int

type TrezorPane struct {
	ui.PaneDescriptor
	*Trezor
	On       bool
	Id       int
	ViewName string
	Mode     string
	Pin      string
	Pass     string

	pin_request  *bus.Message
	pass_request *bus.Message
}

var Status TrezorPane = TrezorPane{
	PaneDescriptor: ui.PaneDescriptor{
		MinWidth:  45,
		MinHeight: 1,
	},
}

func NewTrezorPane(t *Trezor) *TrezorPane {
	TP_Id++
	p := &TrezorPane{
		PaneDescriptor: ui.PaneDescriptor{
			MinWidth:  30,
			MinHeight: 0,
		},
		Id:       TP_Id,
		ViewName: fmt.Sprintf("trezor-%d", TP_Id),
		Trezor:   t,
		On:       true,
	}

	go p.loop()

	return p
}

func (p *TrezorPane) loop() {

	ch := bus.Subscribe(p.ViewName, "timer")
	for msg := range ch {
		if msg.RespondTo != 0 {
			continue // ignore responses
		}
		switch msg.Type {
		case "close":
			return
		case "get_pin":
			if p.pin_request != nil {
				p.pin_request.Respond("", errors.New("canceled"))
			}

			p.pin_request = msg
			p.Pin = ""
			p.setMode("pin")
		case "get_pass":
			if p.pass_request != nil {
				p.pass_request.Respond("", errors.New("canceled"))
			}
			p.pass_request = msg
			p.Pass = ""
			p.setMode("pass")
		case "done": //timer
			id := msg.Data.(int)
			if p.pin_request != nil && p.pin_request.TimerID == id {
				p.pin_request.Respond("", errors.New("timeout"))
				p.pin_request = nil
				p.setMode("")
			}
			if p.pass_request != nil && p.pass_request.TimerID == id {
				p.pass_request.Respond("", errors.New("timeout"))
				p.pass_request = nil
				p.setMode("")
			}
		}
	}
}

func (p *TrezorPane) GetDesc() *ui.PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *TrezorPane) IsOn() bool {
	return p.On
}

func (p *TrezorPane) SetOn(on bool) {
	p.On = on
}

func (p *TrezorPane) EstimateLines(w int) int {
	return gocui.EstimateTemplateLines(p.GetTemplate(), w)
}

func (p *TrezorPane) SetView(x0, y0, x1, y1 int, overlap byte) {

	v, err := ui.Gui.SetView(p.ViewName, x0, y0, x1, y1, overlap)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			log.Error().Err(err).Msgf("SetView error: %s", err)
		}
		p.PaneDescriptor.View = v
		v.JoinedFrame = true
		v.Title = "HW Trezor"
		v.Subtitle = p.Name
		v.Autoscroll = false
		v.ScrollBar = true
		v.OnResize = func(v *gocui.View) {
			v.RenderTemplate(p.GetTemplate())
			v.ScrollTop()
		}
		v.OnOverHotspot = cmn.StandardOnOverHotspot
		v.OnClickHotspot = func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				param := cmn.Split3(hs.Value)

				switch param[0] {
				case "skip_password_on":
					p.Trezor.setSkipPassword(false)
					p.rebuidTemplate()
					return
				case "skip_password_off":
					p.Trezor.setSkipPassword(true)
					p.rebuidTemplate()
					return
				case "button":
					switch param[1] {
					case "back":
						if len(p.Pin) > 0 {
							p.Pin = p.Pin[:len(p.Pin)-1]
							v.GetHotspotById("pin").SetText(strings.Repeat("*", len(p.Pin)) + "______________")
						}
					case "1", "2", "3", "4", "5", "6", "7", "8", "9":
						p.Pin += param[1]
						v.GetHotspotById("pin").SetText(strings.Repeat("*", len(p.Pin)) + "______________")
					case "OK":
						if p.pin_request != nil {
							p.pin_request.Respond(p.Pin, nil)
							p.pin_request = nil
							p.setMode("")
						}
					case "Cancel":
						if p.pin_request != nil {
							p.pin_request.Respond("", errors.New("canceled"))
							p.pin_request = nil
							p.setMode("")
						}
					case "standard":
						if p.pass_request != nil {
							p.pass_request.Respond("", nil)
							p.pass_request = nil
							p.setMode("")
						}
					case "hidden":
						if p.pass_request != nil {
							res := bus.Fetch("timer", "pause", p.pass_request.TimerID)
							if res.Error != nil {
								log.Error().Err(res.Error).Msg("Error pausing timer")
								p.pass_request.Respond("", res.Error)
								p.pass_request = nil
								p.setMode("")
								return
							}
							v.GetGui().ShowPopup(&gocui.Popup{
								Title: "Enter Trezor Password",
								Template: `<c><w>
Password: <input id:password size:16 masked:true>

<button text:OK> <button text:Cancel>`,
								OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
									if hs != nil {
										switch hs.Value {
										case "button OK":
											p.Pass = v.GetInput("password")
											v.GetGui().HidePopup()
											p.pass_request.Respond(p.Pass, nil)
											p.pass_request = nil
											p.setMode("")
										case "button Cancel":
											v.GetGui().HidePopup()
											p.pass_request.Respond("", errors.New("canceled"))
											p.pass_request = nil
											p.setMode("")
										}
									}
								},
								OnClose: func(v *gocui.View) {
									bus.Fetch("timer", "resume", p.pass_request.TimerID)
								},
							})
						}
					}
				}
			}
			cmn.StandardOnClickHotspot(v, hs)
		}

		p.rebuidTemplate()
	}
}

func (p *TrezorPane) setMode(mode string) {
	p.Mode = mode

	if mode != "" {
		p.View.EmFgColor = ui.Gui.ActionBgColor
	} else {
		p.View.EmFgColor = ui.Gui.HelpFgColor
	}

	p.rebuidTemplate()
}

func (p *TrezorPane) rebuidTemplate() {
	temp := ""
	switch p.Mode {
	case "pin":
		temp += "<c>\n"
		temp += "<b>Enter PIN: </b><l id:pin text:'____________'> <button text:'\U000f006e ' id:back>\n\n"

		ids := []int{7, 8, 9, 4, 5, 6, 1, 2, 3}

		for i := 0; i < 9; i++ {
			temp += fmt.Sprintf("<button color:g.HelpFgColor bgcolor:g.HelpBgColor text:' - ' id:%d> ", ids[i])
			if (i+1)%3 == 0 {
				temp += "\n\n"
			}
		}
		temp += "<button text:OK> <button text:Cancel>\n"
	case "pass":
		temp += `<c><w>
<button text:Standard color:g.HelpFgColor bgcolor:g.HelpBgColor id:standard> <button text:Hidden color:g.HelpFgColor bgcolor:g.HelpBgColor id:hidden> 

<button text:Cancel>`
	default:
		if p.Trezor != nil {
			temp += fmt.Sprintf("<b>      SN:</b> %s\n",
				cmn.TagLink(*p.Trezor.DeviceId, "copy "+*p.Trezor.DeviceId, "Copy SN"))

			temp += fmt.Sprintf("<b>Firmware:</b> %d.%d.%d\n",
				*p.Trezor.MajorVersion,
				*p.Trezor.MinorVersion,
				*p.Trezor.PatchVersion)

			if cmn.CurrentWallet != nil {
				temp += "<b>Features:</b> "
				if !p.Trezor.isSkipPassword() {
					temp += cmn.TagLink(gocui.ICON_CHECK, "skip_password_off", "Set passphrase off")
				} else {
					temp += cmn.TagLink(gocui.ICON_UNCHECK, "skip_password_on", "Set passphrase on")
				}
				temp += "Use Passphrase"
			}
		}
	}

	p.SetTemplate(temp)

	ui.Gui.Update(func(g *gocui.Gui) error {
		if p.View != nil {
			p.View.RenderTemplate(p.GetTemplate())
		}
		return nil
	})
}
