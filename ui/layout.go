package ui

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/atotto/clipboard"
	"github.com/rs/zerolog/log"
)

var TopLeftFlow = NewFlow(
	FlowVertical,
	[]Pane{
		&Status,
	},
	&PaneDescriptor{
		MinWidth:  45,
		MinHeight: 1,
	})

var panes = []Pane{
	TopLeftFlow,
	&HailPane,
	&App,
	&LP_V3,
	&Terminal,
}

func Layout(g *gocui.Gui) error {
	panesMutex.Lock() // prevent changing panes while layouting
	defer panesMutex.Unlock()

	maxX, maxY := g.Size()
	pane_map := map[*PaneDescriptor]Pane{}

	// grid layout
	pd := []*PaneDescriptor{}
	for _, p := range panes {
		d := p.GetDesc()
		pane_map[d] = p
		if p.IsOn() {
			pd = append(pd, d)
		}
	}

	// spit to rows
	rows := [][]*PaneDescriptor{}
	row := []*PaneDescriptor{}
	row_min_width := 0
	for _, d := range pd {
		if len(row) == 0 {
			row = append(row, d)
			row_min_width = d.MinWidth
			continue
		}

		if row_min_width+d.MinWidth > maxX {
			rows = append(rows, row)
			row = []*PaneDescriptor{d}
			row_min_width = d.MinWidth
		} else {
			row = append(row, d)
			row_min_width += d.MinWidth
		}
	}

	if len(row) > 0 {
		rows = append(rows, row)
	}

	if len(rows) < 1 {
		log.Error().Msg("No panes to layout")
	}

	row_y := 0
	hight_left := maxY - 1 // minus the bottom bar

	for i, row := range rows {
		last_row := i == len(rows)-1

		widths := make([]int, len(row))
		if len(row) == 1 {
			widths[0] = maxX
		} else {
			// distribute the width
			total := 0
			n_not_fixed := 0
			for i, d := range row {
				widths[i] = d.MinWidth
				total += d.MinWidth
				if !d.fixed_width {
					n_not_fixed++
				}
			}

			// add the rest to the not fixed
			if total < maxX {
				to_add := maxX - total
				left := to_add
				i_nf := 0
				for i, d := range row {
					if !d.fixed_width {
						s := to_add / n_not_fixed
						if i_nf == n_not_fixed-1 {
							s = left
						}
						widths[i] += s
						left -= s
						i_nf++
					}
				}
			}
		}

		// determine the height of the row
		row_height := 0
		if last_row {
			row_height = hight_left
		} else {
			for i, d := range row {
				n_lines := pane_map[d].EstimateLines(widths[i] - 1)
				if n_lines > 0 {
					height := n_lines + 1
					if d.MaxHeight > 0 {
						height = min(d.MaxHeight, height)
					}
					row_height = max(row_height, height)
				} else {
					row_height = max(row_height, d.MinHeight)
				}
			}

			if row_height > hight_left {
				row_height = hight_left
			}

			if hight_left-row_height < 5 { // at least 5 for the last row
				row_height = 0
			} else {

				hight_left -= row_height
			}
		}

		// set the views
		x0 := 0
		for j, d := range row {

			var overlap byte
			if i < len(rows)-1 && len(rows) > 1 {
				overlap |= gocui.BOTTOM
			}

			if i > 0 {
				overlap |= gocui.TOP
			}

			if j < len(row)-1 && len(row) > 1 {
				overlap |= gocui.RIGHT
			}

			if j > 0 {
				overlap |= gocui.LEFT
			}

			if row_height == 0 {
				pane_map[d].SetView(0, 0, 1, 1, overlap)
			} else {
				x1 := x0 + widths[j] - 1
				ox := 0
				if overlap&gocui.RIGHT != 0 {
					ox = 1
				}
				y1 := row_y + row_height - 1
				oy := 0
				if overlap&gocui.BOTTOM != 0 {
					oy = 1
				}
				pane_map[d].SetView(x0, row_y, x1+ox, y1+oy, overlap)
				x0 = x1 + 1
			}
		}

		row_y += row_height

	}

	//	Terminal.SetView(g, 0, FirstRowHeight, maxX-1, maxY-2)
	Bottom.SetView(g)
	Notification.SetView(g)

	g.Cursor = true

	if !Is_ready {
		Is_ready_wg.Done()
		Is_ready = true
	}

	return nil
}

func ProcessOnClickHotspot(v *gocui.View, hs *gocui.Hotspot) {
	index := strings.Index(hs.Value, " ")

	if index == -1 {
		return
	}

	command := hs.Value[:index]
	param := hs.Value[index+1:]

	switch command {
	case "copy":
		clipboard.WriteAll(param)
		Notification.Show("Copied: " + param)
	case "command":
		bus.Send("ui", "command", param)
	case "start_command":
		bus.Send("ui", "start_command", param)
	case "open":
		cmn.OpenBrowser(param)
	case "system":
		cmn.SystemCommand(param)
	}
}

func ProcessOnOverHotspot(v *gocui.View, hs *gocui.Hotspot) {
	if hs != nil {
		Bottom.Printf(hs.Tip)
	} else {
		Bottom.Printf("")
	}
}
