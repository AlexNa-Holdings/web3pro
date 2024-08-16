package ui

import (
	"sync"

	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/rs/zerolog/log"
)

var panesMutex = &sync.Mutex{}

type Pane interface {
	SetView(int, int, int, int)
	GetDesc() *PaneDescriptor
	GetTemplate() string
}

type PaneDescriptor struct {
	On          bool
	MinWidth    int
	MinHeight   int
	MaxHeight   int
	MaxWidth    int
	fixed_width bool
	View        *gocui.View
}

func (p *PaneDescriptor) ShowPane() {
	panesMutex.Lock()
	defer panesMutex.Unlock()

	p.On = true
}

func (p *PaneDescriptor) HidePane() {
	panesMutex.Lock()
	defer panesMutex.Unlock()

	p.On = false

	if p.View != nil {
		Gui.DeleteView(p.View.Name())
		log.Debug().Msgf("Deleted view %s", p.View.Name())
		p.View = nil
	}
}
