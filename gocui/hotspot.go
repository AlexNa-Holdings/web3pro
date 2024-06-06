package gocui

import (
	"errors"
	"fmt"
	"sort"
)

type Hotspot struct {
	X, Y            int
	L               int
	Value           string
	Tip             string
	Cells           []cell
	CellsHighligted []cell
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

	// Find the index where the new hotspot should be inserted
	index := sort.Search(len(v.hotspots), func(i int) bool {
		if v.hotspots[i].Y == y {
			return v.hotspots[i].X >= x
		}
		return v.hotspots[i].Y >= y
	})

	h := Hotspot{x, y, len(cells), value, tip, cells, cells_highligted}

	// Insert the new hotspot at the found index
	v.hotspots = append(v.hotspots, h)             // Increase the size by one
	copy(v.hotspots[index+1:], v.hotspots[index:]) // Shift elements to the right
	v.hotspots[index] = h                          // Insert the new element

	return nil
}

func (v *View) findHotspot(x, y int) *Hotspot {

	if v.hotspots == nil {
		return nil
	}

	// Use binary search to find the first hotspot on line N
	i := sort.Search(len(v.hotspots), func(i int) bool {
		return v.hotspots[i].Y >= y
	})

	for ; i < len(v.hotspots) && v.hotspots[i].Y == y; i++ {
		h := v.hotspots[i]
		if x >= h.X && x < h.X+h.L {
			return &h
		}
	}

	return nil
}

func (v *View) AddLink(text, value, tip string) error {
	cells := AddCells(nil, v.gui.EmFgColor, v.BgColor, text)
	cells_highligted := AddCells(nil, v.SelFgColor, v.SelBgColor, text)
	err := v.AddHotspot(v.wx, v.wy, value, tip, cells, cells_highligted)
	fmt.Fprint(v, text)
	return err

}
