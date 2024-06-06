package ui

import (
	"errors"
	"fmt"
	"time"

	"github.com/AlexNa-Holdings/web3pro/gocui"
)

const NOTIFICATION_TIME = 3

var time_to_hide_notification time.Time

type NotificationPane struct {
	*gocui.View
	On      bool
	Message string
}

var Notification *NotificationPane = &NotificationPane{}

func (p *NotificationPane) SetView(g *gocui.Gui) {
	if !p.On {
		return
	}

	var err error
	maxX, maxY := g.Size()

	x := maxX - len(p.Message) - 1
	if x < 0 {
		x = 0
	}

	if p.View, err = g.SetView("notifiction", x, maxY-2, maxX, maxY, 0); err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			panic(err)
		}
		p.View.Autoscroll = false
		p.View.Frame = false
		p.View.BgColor = CurrentTheme.HelpBgColor
		p.View.FgColor = CurrentTheme.HelpFgColor
		fmt.Fprintf(p.View, p.Message)
	}
}

func (n *NotificationPane) Show(text string) {
	n.Hide()
	Gui.Update(func(g *gocui.Gui) error {
		n.Message = text
		n.On = true
		return nil
	})

	time_to_hide_notification = time.Now().Add(NOTIFICATION_TIME * time.Second)
	go func() {
		time.Sleep(NOTIFICATION_TIME * time.Second)
		if time.Now().After(time_to_hide_notification) {
			n.Hide()
		}
	}()
}

func (n *NotificationPane) Hide() {
	if n.View != nil {
		n.View.Clear()
	}

	Gui.Update(func(g *gocui.Gui) error {
		Gui.DeleteView("notifiction")
		n.On = false
		return nil
	})

}
