package ui

import (
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/atotto/clipboard"
	"github.com/rs/zerolog/log"
)

var panes = []Pane{
	&Status,
	&HailPane,
	&Terminal,
}

func Layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	pane_map := map[*PaneDescriptor]Pane{}

	// grid layout
	pd := []*PaneDescriptor{}
	for _, p := range panes {
		d := p.GetDesc()
		pane_map[d] = p
		if d.On {
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
				template := pane_map[d].GetTemplate()
				if template != "" {
					n_lines := gocui.EstimateTemplateLines(template, widths[i]-1)
					row_height = max(row_height, n_lines+2)
				} else {
					row_height = max(row_height, d.MinHeight)
				}
			}

			if row_height > hight_left {
				row_height = hight_left
			}

			if hight_left-row_height < 5 { // at least 5 for the last row
				row_height = 0
				log.Error().Msg("Not enough space for the last row")
			} else {

				hight_left -= row_height
			}
		}

		// set the views
		x0 := 0
		for i, d := range row {
			if row_height == 0 {
				pane_map[d].SetView(0, 0, 1, 1)
			} else {
				x1 := x0 + widths[i] - 1
				y1 := row_y + row_height - 1
				pane_map[d].SetView(x0, row_y, x1, y1)
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
	case "open":
		cmn.OpenBrowser(param)
	}
}

func ProcessOnOverHotspot(v *gocui.View, hs *gocui.Hotspot) {
	if hs != nil {
		Bottom.Printf(hs.Tip)
	} else {
		Bottom.Printf("")
	}
}
