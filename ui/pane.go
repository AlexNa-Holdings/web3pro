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
	MinWidth               int
	MinHeight              int
	MaxHeight              int
	MaxWidth               int
	fixed_width            bool
	View                   *gocui.View
	SupportCachedHightCalc bool

	_template           string
	_calculated_heights map[int]int
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

func (d *PaneDescriptor) GetTemplate() string {
	return d._template
}

func (d *PaneDescriptor) SetTemplate(t string) {
	d._calculated_heights = make(map[int]int)
	d._template = t
}

func FastEstimateLines(p Pane, w int) int {

	//	return p.EstimateLines(w) // DEBUG

	d := p.GetDesc()

	if !d.SupportCachedHightCalc {
		return p.EstimateLines(w)
	}

	if d._calculated_heights == nil {
		d._calculated_heights = make(map[int]int)
	}

	l, ok := d._calculated_heights[w]
	if ok {
		return l
	}
	l = p.EstimateLines(w)
	d._calculated_heights[w] = l
	return l
}
