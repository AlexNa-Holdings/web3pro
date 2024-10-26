package ui

type Orientation int

const (
	FlowHorizontal Orientation = iota
	FlowVertical
)

type PaneFlow struct {
	PaneDescriptor
	Orientation Orientation
	Panes       []Pane
}

func NewFlow(o Orientation, panes []Pane, descr *PaneDescriptor) *PaneFlow {

	if descr == nil {
		descr = &PaneDescriptor{
			MinWidth:               30,
			MinHeight:              1,
			MaxHeight:              20,
			SupportCachedHightCalc: false,
		}
	}

	return &PaneFlow{
		PaneDescriptor: *descr,
		Orientation:    o,
		Panes:          panes,
	}
}

func (p *PaneFlow) IsOn() bool {

	for _, pane := range p.Panes {
		if pane.IsOn() {
			return true
		}
	}

	return false
}

func (p *PaneFlow) SetOn(on bool) {
}

func (p *PaneFlow) EstimateLines(w int) int {

	n_lines := 0

	for i, pane := range p.Panes {
		if !pane.IsOn() {
			continue
		}

		if p.Orientation == FlowHorizontal {
			widths := p.spread_horizontaly(w)
			n_lines = max(n_lines, pane.EstimateLines(widths[i]))
		} else {
			n_lines += pane.EstimateLines(w)
		}
	}

	return n_lines
}

func (p *PaneFlow) GetDesc() *PaneDescriptor {
	return &p.PaneDescriptor
}

func (p *PaneFlow) spread_horizontaly(width int) []int {
	ws := make([]int, len(p.Panes))

	total_min_width := 0
	for _, pane := range p.Panes {
		if pane.IsOn() {
			total_min_width += pane.GetDesc().MinWidth
		}
	}

	last := 0

	if total_min_width > width {
		// distribute the width
		total := 0
		n_not_fixed := 0
		for i, d := range p.Panes {
			if !d.IsOn() {
				continue
			}
			d := d.GetDesc()
			ws[i] = d.MinWidth
			total += d.MinWidth
			last = i
			if !d.fixed_width {
				n_not_fixed++
			}
		}

		// add the rest to the not fixed
		if total < width {
			to_add := width - total
			left := to_add
			i_nf := 0
			for i, p := range p.Panes {
				if !p.IsOn() {
					continue
				}
				d := p.GetDesc()
				if !d.fixed_width {
					s := to_add / n_not_fixed
					ws[i] += s
					left -= s
					i_nf++
					last = i
				}
			}

			ws[last] += left
		}
	} else {
		left := width
		last := 0
		for i, p := range p.Panes {
			if !p.IsOn() {
				continue
			}
			d := p.GetDesc()
			ws[i] = min(d.MinWidth, left)
			left -= ws[i]
			last = i
		}

		ws[last] += left
	}

	return ws
}

func (p *PaneFlow) spread_vertically(width, height int) []int {

	hs := make([]int, len(p.Panes))
	total_height := 0
	last := 0

	for i, pane := range p.Panes {
		if pane.IsOn() {
			hs[i] = pane.EstimateLines(width)
			total_height += hs[i]
			last = i
		} else {
			hs[i] = 0
		}
	}

	if total_height <= height {
		// add the rest to the last
		hs[last] += height - total_height
		return hs
	} else {
		// distribute the width proportionally
		total := 0
		for i := 0; i < len(hs); i++ {
			hs[i] = hs[i] * height / total_height
			total += hs[i]
			last = i
		}

		hs[last] += height - total
	}

	return hs
}

func (p *PaneFlow) SetView(x0, y0, x1, y1 int, overlap byte) {

	if len(p.Panes) == 0 {
		return
	}

	if p.Orientation == FlowHorizontal {
		widths := p.spread_horizontaly(x1 - x0)

		x := x0
		for i, pane := range p.Panes {
			if pane.IsOn() {
				pane.SetView(x, y0, x+widths[i], y1, overlap)
				x += widths[i]
			} else {
				if pane.GetDesc().View != nil {
					Gui.DeleteView(pane.GetDesc().View.Name())
					pane.GetDesc().View = nil
				}
			}
		}
	} else {
		heights := p.spread_vertically(x1-x0, y1-y0)

		y := y0
		for i, pane := range p.Panes {
			if pane.IsOn() {
				pane.SetView(x0, y, x1, y+heights[i], overlap)
				y += heights[i]
			} else {
				if pane.GetDesc().View != nil {
					Gui.DeleteView(pane.GetDesc().View.Name())
					pane.GetDesc().View = nil
				}
			}
		}
	}

}

func (p *PaneFlow) AddPane(pane Pane) {
	p.Panes = append(p.Panes, pane)
	Flush()
}

func (p *PaneFlow) RemovePane(pane Pane) {
	for i, pi := range p.Panes {
		if pi == pane {
			Gui.DeleteView(pi.GetDesc().View.Name())
			pi.GetDesc().View = nil
			p.Panes = append(p.Panes[:i], p.Panes[i+1:]...)
			return
		}
	}
	Flush()
}
