package ui

import (
	"sync"

	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

var panesMutex = &sync.Mutex{}

type Pane interface {
	SetView(int, int, int, int, byte)
	GetDesc() *PaneDescriptor
	EstimateLines(int) int
	IsOn() bool
	SetOn(bool)
}

type PaneDescriptor struct {
	MinWidth    int
	MinHeight   int
	MaxHeight   int
	MaxWidth    int
	fixed_width bool
	View        *gocui.View
}

func ShowPane(p Pane) {
	panesMutex.Lock()
	defer panesMutex.Unlock()

	p.SetOn(true)
}

func HidePane(p Pane) {
	panesMutex.Lock()
	defer panesMutex.Unlock()

	p.SetOn(false)

	d := p.GetDesc()

	if d.View != nil {
		Gui.DeleteView(d.View.Name())
		log.Debug().Msgf("Deleted view %s", d.View.Name())
		d.View = nil
	}

	Flush()
}
