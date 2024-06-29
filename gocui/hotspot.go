package gocui

import (
	"errors"
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

func (v *View) AddHotspot(x, y int, value string, tip string, cells []cell, cells_highligted []cell) (*Hotspot, error) {
	if len(cells) != len(cells_highligted) {
		return nil, errors.New("cells and cells_highligted must have the same length")
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
	v.hotspots = append(v.hotspots, &h)            // Increase the size by one
	copy(v.hotspots[index+1:], v.hotspots[index:]) // Shift elements to the right
	v.hotspots[index] = &h                         // Insert the new element

	return &h, nil
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
			return h
		}
	}

	return nil
}
