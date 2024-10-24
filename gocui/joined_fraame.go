package gocui

func (g *Gui) drawJoinedFrame() error {

	maxX, maxY := screen.Size()
	var intersections = make(map[int]byte)

	if len(g.JoinedFrameRunes) < 11 {
		return nil
	}

	fg, bg := g.JoinedFrameFgColor, g.JoinedFrameBgColor

	for _, v := range g.views {
		if !v.JoinedFrame || !v.Visible || v.y0 >= maxY || v.x0 >= maxX {
			continue
		}

		for x := v.x0 + 1; x < v.x1 && x < g.maxX; x++ {
			if x < 0 {
				continue
			}
			if v.y0 > -1 && v.y0 < g.maxY {
				xy := XY(x, v.y0)
				intersections[xy] = intersections[xy] | LEFT | RIGHT
			}
			if v.y1 > -1 && v.y1 < g.maxY {
				xy := XY(x, v.y1)
				intersections[xy] = intersections[xy] | LEFT | RIGHT
			}
		}
		for y := v.y0 + 1; y < v.y1 && y < g.maxY; y++ {
			if y < 0 {
				continue
			}
			if v.x0 > -1 && v.x0 < g.maxX {
				xy := XY(v.x0, y)
				intersections[xy] = intersections[xy] | TOP | BOTTOM
			}
			if v.x1 > -1 && v.x1 < g.maxX {
				xy := XY(v.x1, y)
				intersections[xy] = intersections[xy] | TOP | BOTTOM
			}
		}

		intersections[XY(v.x0, v.y0)] = intersections[XY(v.x0, v.y0)] | RIGHT | BOTTOM
		intersections[XY(v.x1, v.y0)] = intersections[XY(v.x1, v.y0)] | LEFT | BOTTOM
		intersections[XY(v.x0, v.y1)] = intersections[XY(v.x0, v.y1)] | RIGHT | TOP
		intersections[XY(v.x1, v.y1)] = intersections[XY(v.x1, v.y1)] | LEFT | TOP

	}

	//{'─', '│', '┌', '┐', '└', '┘', '├', '┤', '┬', '┴', '┼'}

	for xy, intersect := range intersections {
		x, y := xy&0xffff, xy>>16

		switch intersect {
		case LEFT | RIGHT:
			g.SetRune(x, y, g.JoinedFrameRunes[0], fg, bg)
		case TOP | BOTTOM:
			g.SetRune(x, y, g.JoinedFrameRunes[1], fg, bg)
		case RIGHT | BOTTOM:
			g.SetRune(x, y, g.JoinedFrameRunes[2], fg, bg)
		case LEFT | BOTTOM:
			g.SetRune(x, y, g.JoinedFrameRunes[3], fg, bg)
		case RIGHT | TOP:
			g.SetRune(x, y, g.JoinedFrameRunes[4], fg, bg)
		case LEFT | TOP:
			g.SetRune(x, y, g.JoinedFrameRunes[5], fg, bg)
		case TOP | BOTTOM | RIGHT:
			g.SetRune(x, y, g.JoinedFrameRunes[6], fg, bg)
		case TOP | BOTTOM | LEFT:
			g.SetRune(x, y, g.JoinedFrameRunes[7], fg, bg)
		case LEFT | RIGHT | BOTTOM:
			g.SetRune(x, y, g.JoinedFrameRunes[8], fg, bg)
		case LEFT | RIGHT | TOP:
			g.SetRune(x, y, g.JoinedFrameRunes[9], fg, bg)
		case LEFT | RIGHT | TOP | BOTTOM:
			g.SetRune(x, y, g.JoinedFrameRunes[10], fg, bg)
		}
	}
	return nil
}

func XY(x, y int) int {
	return (x & 0xffff) | (y&0xffff)<<16
}
