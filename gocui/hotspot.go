package gocui

import (
	"errors"
	"sort"
)

type hotspot struct {
	x, y             int
	l                int
	value            string
	tip              string
	cells            []cell
	cells_highligted []cell
}

func AddCells(cells []cell, fg, bg Attribute, text string) []cell {
	if cells == nil {
		cells = []cell{}
	}
	for _, r := range text {
		cells = append(cells, cell{r, bg, fg})
	}
	return cells
}

func (v *View) AddHotspot(x, y int, value string, tip string, cells []cell, cells_highligted []cell) error {

	if len(cells) != len(cells_highligted) {
		return errors.New("cells and cells_highligted must have the same length")
	}

	//insert into hotspots sorted by x and y
	h := hotspot{x, y, len(cells), value, tip, cells, cells_highligted}
	var p int
	var hs hotspot
	for p, hs = range v.hotspots {
		if hs.y > y || (hs.y == y && hs.x > x) {
			break
		}
	}

	v.hotspots = append(v.hotspots[:p], append([]hotspot{h}, v.hotspots[p:]...)...)

	return nil
}

func (v *View) findHotspot(x, y int) *hotspot {

	if v.hotspots == nil {
		return nil
	}

	// Use binary search to find the first hotspot on line N
	i := sort.Search(len(v.hotspots), func(i int) bool {
		return v.hotspots[i].y >= y
	})

	for ; i < len(v.hotspots) && v.hotspots[i].y == y; i++ {
		h := v.hotspots[i]
		if x >= h.x && x < h.x+h.l {
			return &h
		}
	}

	return nil
}

func (v *View) AddLink(text, value, tip string) error {
	cells := AddCells(nil, v.gui.EmFgColor, v.BgColor, text)
	cells_highligted := AddCells(nil, v.SelFgColor, v.SelBgColor, text)
	v.writeCells(v.wx, v.wy, cells)
	return v.AddHotspot(v.wx, v.wy, value, tip, cells, cells_highligted)
}
