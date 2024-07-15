// Copyright 2014 The gocui Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gocui

import (
	"bytes"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/rs/zerolog/log"
)

// Constants for overlapping edges
const (
	TOP    = 1 // view is overlapping at top edge
	BOTTOM = 2 // view is overlapping at bottom edge
	LEFT   = 4 // view is overlapping at left edge
	RIGHT  = 8 // view is overlapping at right edge
)

const (
	ICON_DELETE   = "\U0000f057 " //"\uf00d"
	ICON_EDIT     = "\uf044 "
	ICON_COPY     = "\uf0c5 "
	ICON_DROPLIST = "\ueb6e "
	ICON_PROMOTE  = "\ued65 "
	ICON_ADD      = "\ueadc "
	ICON_3DOTS    = "\U000f01d8"
	ICON_BACK     = "\U000f006e "
	ICON_SEND     = "\U000f048a "
	ICON_LINK     = "\uf08e "
	ICON_FEED     = "\uf09e "
)

const REGEX_TAGS = `<(/?\w+)((?:\s+\w+(?::(?:[^>\s]+|"[^"]*"|'[^']*'))?\s*)*)>`
const REGEX_SINGLE_TAG = `(/?\w+)(?::(".*?"|'.*?'|[^>\s]+))`

var (
	// ErrInvalidPoint is returned when client passed invalid coordinates of a cell.
	// Most likely client has passed negative coordinates of a cell.
	ErrInvalidPoint = errors.New("invalid point")
)

const (
	C_LINK = iota
	C_BUTTON
	C_INPUT
	C_TEXT_INPUT
	C_SELECT
)

type PopoupControl struct {
	Type           PUCType
	ID             string
	x0, y0, x1, y1 int
	*View
	*Hotspot
	Items []string
	Value string
}

// A View is a window. It maintains its own internal buffer and cursor
// position.
type View struct {
	name           string
	x0, y0, x1, y1 int              // left top right bottom
	ox, oy         int              // view offsets
	cx, cy         int              // cursor position
	rx, ry         int              // Read() offsets
	wx, wy         int              // Write() offsets
	lines          [][]cell         // All the data
	hotspots       []*Hotspot       // AN - hotspots sorted by positions
	Controls       []*PopoupControl // AN - controls
	ControlInFocus int              // AN - the control in focus

	outMode OutputMode

	activeHotspot  *Hotspot                   // AN - the currently active hotspot
	OnOverHotspot  func(v *View, hs *Hotspot) // AN - function to be called when the mouse is over a hotspot
	OnClickHotspot func(v *View, hs *Hotspot) // AN - function to be called when the mouse is clicked on a hotspot
	OnClickTitle   func(v *View)              // AN - function to be called when the title is clicked
	OnResize       func(v *View)              // AN - function to be called when the view is resized
	DropList       *View                      // AN - the view that is used for the combo list

	// readBuffer is used for storing unread bytes
	readBuffer []byte

	// tained is true if the viewLines must be updated
	tainted bool

	// contentCache is the content the frame
	// if a redraw is request with tainted is false this will be used to draw the frame
	contentCache []cellCache

	// writeMutex protects locks the write process
	writeMutex sync.Mutex

	// ei is used to decode ESC sequences on Write
	ei *escapeInterpreter

	// Visible specifies whether the view is visible.
	Visible bool

	// AN Scrool Bar support
	ScrollBar bool

	ScrollBarStatus struct {
		shown    bool
		height   int
		position int
	}

	// BgColor and FgColor allow to configure the background and foreground
	// colors of the View.
	BgColor, FgColor Attribute

	// SelBgColor and SelFgColor are used to configure the background and
	// foreground colors of the selected line, when it is highlighted.
	SelBgColor, SelFgColor Attribute

	//A.N.
	SubTitleFgColor Attribute
	SubTitleBgColor Attribute
	EmFgColor       Attribute
	TitleAttrib     Attribute
	SubTitleAttrib  Attribute

	// If Editable is true, keystrokes will be added to the view's internal
	// buffer at the cursor position.
	Editable bool

	// Editor allows to define the editor that manages the editing mode,
	// including keybindings or cursor behaviour. DefaultEditor is used by
	// default.
	Editor Editor

	// Overwrite enables or disables the overwrite mode of the view.
	Overwrite bool

	// If Highlight is true, Sel{Bg,Fg}Colors will be used
	// for the line under the cursor position.
	Highlight bool

	// If Frame is true, a border will be drawn around the view.
	Frame bool

	// FrameColor allow to configure the color of the Frame when it is not highlighted.
	FrameColor Attribute

	// FrameRunes allows to define custom runes for the frame edges.
	// The rune slice can be defined with 3 different lengths.
	// If slice doesn't match these lengths, default runes will be used instead of missing one.
	//
	// 2 runes with only horizontal and vertical edges.
	//  []rune{'─', '│'}
	//  []rune{'═','║'}
	// 6 runes with horizontal, vertical edges and top-left, top-right, bottom-left, bottom-right cornes.
	//  []rune{'─', '│', '┌', '┐', '└', '┘'}
	//  []rune{'═','║','╔','╗','╚','╝'}
	// 11 runes which can be used with `gocui.Gui.SupportOverlaps` property.
	//  []rune{'─', '│', '┌', '┐', '└', '┘', '├', '┤', '┬', '┴', '┼'}
	//  []rune{'═','║','╔','╗','╚','╝','╠','╣','╦','╩','╬'}
	FrameRunes []rune

	// If Wrap is true, the content that is written to this View is
	// automatically wrapped when it is longer than its width. If true the
	// view's x-origin will be ignored.
	Wrap bool

	// If Autoscroll is true, the View will automatically scroll down when the
	// text overflows. If true the view's y-origin will be ignored.
	Autoscroll bool

	// If Frame is true, Title allows to configure a title for the view.
	Title string

	// TitleColor allow to configure the color of title and subtitle for the view.
	TitleColor Attribute

	// If Frame is true, Subtitle allows to configure a subtitle for the view.
	Subtitle string

	// If Mask is true, the View will display the mask instead of the real
	// content
	Mask rune

	// Overlaps describes which edges are overlapping with another view's edges
	Overlaps byte

	// If HasLoader is true, the message will be appended with a spinning loader animation
	HasLoader bool

	// KeybindOnEdit should be set to true when you want to execute keybindings even when the view is editable
	// (this is usually not the case)
	KeybindOnEdit bool

	// gui contains the view it's gui
	gui *Gui
}

type cell struct {
	chr              rune
	bgColor, fgColor Attribute
}

type cellCache struct {
	chr              rune
	bgColor, fgColor Attribute
	x, y             int
}

type lineType []cell

func (v *View) GetGui() *Gui {
	return v.gui
}

// String returns a string from a given cell slice.
func (l lineType) String() string {
	str := ""
	for _, c := range l {
		str += string(c.chr)
	}
	return str
}

// newView returns a new View object.
func (g *Gui) newView(name string, x0, y0, x1, y1 int, mode OutputMode) *View {
	v := &View{
		name:           name,
		x0:             x0,
		y0:             y0,
		x1:             x1,
		y1:             y1,
		Visible:        true,
		Frame:          true,
		Editor:         DefaultEditor,
		tainted:        true,
		outMode:        mode,
		ei:             newEscapeInterpreter(mode),
		gui:            g,
		ControlInFocus: -1,
		TitleAttrib:    AttrBold,
	}

	// v.FgColor, v.BgColor = ColorDefault, ColorDefault
	// v.SelFgColor, v.SelBgColor = ColorDefault, ColorDefault
	// v.TitleColor, v.FrameColor = ColorDefault, ColorDefault

	v.FgColor = v.gui.FgColor
	v.BgColor = v.gui.BgColor
	v.SelFgColor = v.gui.SelFgColor
	v.SelBgColor = v.gui.SelBgColor
	v.FrameColor = v.gui.FrameColor
	v.TitleColor = v.gui.FrameColor
	v.EmFgColor = v.gui.EmFgColor
	v.SubTitleFgColor = v.gui.SubTitleFgColor
	v.SubTitleBgColor = v.gui.SubTitleBgColor

	return v
}

// Dimensions returns the dimensions of the View
func (v *View) Dimensions() (int, int, int, int) {
	return v.x0, v.y0, v.x1, v.y1
}

// Size returns the number of visible columns and rows in the View.
func (v *View) Size() (x, y int) {
	return v.x1 - v.x0 - 1, v.y1 - v.y0 - 1
}

// Name returns the name of the view.
func (v *View) Name() string {
	return v.name
}

// setRune sets a rune at the given point relative to the view. It applies the
// specified colors, taking into account if the cell must be highlighted. Also,
// it checks if the position is valid.
func (v *View) setRune(x, y int, ch rune, fgColor, bgColor Attribute) error {
	maxX, maxY := v.Size()
	if x < 0 || x >= maxX || y < 0 || y >= maxY {
		return ErrInvalidPoint
	}

	if v.Mask != 0 {
		fgColor = v.FgColor
		bgColor = v.BgColor
		ch = v.Mask
	} else if v.Highlight && y == v.cy-v.oy {
		fgColor = v.SelFgColor | AttrBold
		bgColor = v.SelBgColor | AttrBold
	} else if hs := v.findHotspot(x+v.ox, y+v.oy); hs != nil {
		hsx := x + v.ox - hs.X

		if (v.ControlInFocus != -1 && v.Controls[v.ControlInFocus].Hotspot == hs) ||
			(y == v.cy-v.oy &&
				hsx >= 0 && hsx < hs.L &&
				v.cx+v.ox >= hs.X &&
				v.cx+v.ox < hs.X+hs.L) {
			fgColor = hs.CellsHighligted[hsx].fgColor
			bgColor = hs.CellsHighligted[hsx].bgColor
			ch = hs.CellsHighligted[hsx].chr
		} else {
			fgColor = hs.Cells[hsx].fgColor
			bgColor = hs.Cells[hsx].bgColor
			ch = hs.Cells[hsx].chr
		}
	}

	// Don't display NUL characters
	if ch == 0 {
		ch = ' '
	}

	tcellSetCell(v.x0+x+1, v.y0+y+1, ch, fgColor, bgColor, v.outMode)

	return nil
}

// SetCursorUnrestricted sets the cursor position of the view at the given point
// This does NOT check if the x and y location are available in the buffer
//
// Rules:
//
//	y >= 0
//	x >= 0
func (v *View) SetCursorUnrestricted(x, y int) error {
	if x < 0 || y < 0 {
		return ErrInvalidPoint
	}

	v.cx = x
	v.cy = y
	return nil
}

// SetCursor tries sets the cursor position of the view at the given point
// If the x or y are outside of the buffer this function will place the cursor on the nearest buffer location
//
// Rules:
//
//	y >= 0
//	x >= 0
func (v *View) SetCursor(x, y int) error {
	if hs := v.findHotspot(x, y); hs != nil {
		if v.activeHotspot != hs {
			v.activeHotspot = hs
			if v.OnOverHotspot != nil {
				v.OnOverHotspot(v, hs)
			}
		}
	} else {
		if v.activeHotspot != nil {
			v.activeHotspot = nil
			if v.OnOverHotspot != nil {
				v.OnOverHotspot(v, nil)
			}
		}
	}

	if len(v.lines) == 0 {
		y = 0
	} else if y >= len(v.lines) && y != 0 {
		y = len(v.lines) - 1
	}

	if x > 0 && (len(v.lines) == 0 || len(v.lines[y]) < x) {
		if len(v.lines) == 0 {
			x = 0
		} else {
			x = len(v.lines[y])
		}
	}

	return v.SetCursorUnrestricted(x, y)
}

// Cursor returns the cursor position of the view.
func (v *View) Cursor() (x, y int) {
	return v.cx, v.cy
}

// SetOrigin sets the origin position of the view's internal buffer,
// so the buffer starts to be printed from this point, which means that
// it is linked with the origin point of view. It can be used to
// implement Horizontal and Vertical scrolling with just incrementing
// or decrementing ox and oy.
func (v *View) SetOrigin(x, y int) error {
	if x < 0 || y < 0 {
		return ErrInvalidPoint
	}
	v.ox = x
	v.oy = y
	return nil
}

// Origin returns the origin position of the view.
func (v *View) Origin() (x, y int) {
	return v.ox, v.oy
}

// SetWritePos sets the write position of the view's internal buffer.
// So the next Write call would write directly to the specified position.
func (v *View) SetWritePos(x, y int) error {
	if x < 0 || y < 0 {
		return ErrInvalidPoint
	}
	v.wx = x
	v.wy = y
	return nil
}

// WritePos returns the current write position of the view's internal buffer.
func (v *View) WritePos() (x, y int) {
	return v.wx, v.wy
}

// SetReadPos sets the read position of the view's internal buffer.
// So the next Read call would read from the specified position.
func (v *View) SetReadPos(x, y int) error {
	if x < 0 || y < 0 {
		return ErrInvalidPoint
	}
	v.readBuffer = nil
	v.rx = x
	v.ry = y
	return nil
}

// ReadPos returns the current read position of the view's internal buffer.
func (v *View) ReadPos() (x, y int) {
	return v.rx, v.ry
}

// makeWriteable creates empty cells if required to make position (x, y) writeable.
func (v *View) makeWriteable(x, y int) {
	// TODO: make this more efficient

	// line `y` must be index-able (that's why `<=`)
	for len(v.lines) <= y {
		if cap(v.lines) > len(v.lines) {
			newLen := cap(v.lines)
			if newLen > y {
				newLen = y + 1
			}
			v.lines = v.lines[:newLen]
		} else {
			v.lines = append(v.lines, nil)
		}
	}
	// cell `x` must not be index-able (that's why `<`)
	// append should be used by `lines[y]` user if he wants to write beyond `x`
	for len(v.lines[y]) < x {
		if cap(v.lines[y]) > len(v.lines[y]) {
			newLen := cap(v.lines[y])
			if newLen > x {
				newLen = x
			}
			v.lines[y] = v.lines[y][:newLen]
		} else {
			v.lines[y] = append(v.lines[y], cell{})
		}
	}
}

// writeCells copies []cell to specified location (x, y)
// !!! caller MUST ensure that specified location (x, y) is writeable by calling makeWriteable
func (v *View) writeCells(x, y int, cells []cell) {
	var newLen int
	// use maximum len available
	line := v.lines[y][:cap(v.lines[y])]
	maxCopy := len(line) - x
	if maxCopy < len(cells) {
		copy(line[x:], cells[:maxCopy])
		line = append(line, cells[maxCopy:]...)
		newLen = len(line)
	} else { // maxCopy >= len(cells)
		copy(line[x:], cells)
		newLen = x + len(cells)
		if newLen < len(v.lines[y]) {
			newLen = len(v.lines[y])
		}
	}
	v.lines[y] = line[:newLen]
}

// Write appends a byte slice into the view's internal buffer. Because
// View implements the io.Writer interface, it can be passed as parameter
// of functions like fmt.Fprintf, fmt.Fprintln, io.Copy, etc. Clear must
// be called to clear the view's buffer.
func (v *View) Write(p []byte) (n int, err error) {
	v.tainted = true
	v.writeMutex.Lock()
	defer v.writeMutex.Unlock()
	v.makeWriteable(v.wx, v.wy)
	v.writeRunes(bytes.Runes(p))

	return len(p), nil
}

func (v *View) WriteRunes(p []rune) {
	v.tainted = true

	// Fill with empty cells, if writing outside current view buffer
	v.makeWriteable(v.wx, v.wy)
	v.writeRunes(p)
}

func (v *View) WriteString(s string) {
	v.WriteRunes([]rune(s))
}

// writeRunes copies slice of runes into internal lines buffer.
// caller must make sure that writing position is accessable.
func (v *View) writeRunes(p []rune) {
	for _, r := range p {
		switch r {
		case '\n':
			v.wy++
			if v.wy >= len(v.lines) {
				v.lines = append(v.lines, nil)
			}

			fallthrough
			// not valid in every OS, but making runtime OS checks in cycle is bad.
		case '\r':
			v.wx = 0
		default:
			cells := v.parseInput(r)
			if cells == nil {
				continue
			}
			v.writeCells(v.wx, v.wy, cells)
			v.wx += len(cells)
		}
	}
}

// parseInput parses char by char the input written to the View. It returns nil
// while processing ESC sequences. Otherwise, it returns a cell slice that
// contains the processed data.
func (v *View) parseInput(ch rune) []cell {
	cells := []cell{}

	isEscape, err := v.ei.parseOne(ch)
	if err != nil {
		for _, r := range v.ei.runes() {
			c := cell{
				fgColor: v.FgColor,
				bgColor: v.BgColor,
				chr:     r,
			}
			cells = append(cells, c)
		}
		v.ei.reset()
	} else {
		if isEscape {
			return nil
		}
		repeatCount := 1
		if ch == '\t' {
			ch = ' '
			repeatCount = 4
		}
		for i := 0; i < repeatCount; i++ {
			c := cell{
				fgColor: v.ei.curFgColor,
				bgColor: v.ei.curBgColor,
				chr:     ch,
			}
			cells = append(cells, c)
		}
	}

	return cells
}

// Read reads data into p from the current reading position set by SetReadPos.
// It returns the number of bytes read into p.
// At EOF, err will be io.EOF.
func (v *View) Read(p []byte) (n int, err error) {
	buffer := make([]byte, utf8.UTFMax)
	offset := 0
	if v.readBuffer != nil {
		copy(p, v.readBuffer)
		if len(v.readBuffer) >= len(p) {
			if len(v.readBuffer) > len(p) {
				v.readBuffer = v.readBuffer[len(p):]
			}
			return len(p), nil
		}
		v.readBuffer = nil
	}
	for v.ry < len(v.lines) {
		for v.rx < len(v.lines[v.ry]) {
			count := utf8.EncodeRune(buffer, v.lines[v.ry][v.rx].chr)
			copy(p[offset:], buffer[:count])
			v.rx++
			newOffset := offset + count
			if newOffset >= len(p) {
				if newOffset > len(p) {
					v.readBuffer = buffer[newOffset-len(p):]
				}
				return len(p), nil
			}
			offset += count
		}
		v.rx = 0
		v.ry++
	}
	return offset, io.EOF
}

// Rewind sets read and write pos to (0, 0).
func (v *View) Rewind() {
	if err := v.SetReadPos(0, 0); err != nil {
		// SetReadPos returns error only if x and y are negative
		// we are passing 0, 0, thus no error should occur.
		log.Error().Err(err).Msgf("SetView error: %s", err)
	}
	if err := v.SetWritePos(0, 0); err != nil {
		// SetWritePos returns error only if x and y are negative
		// we are passing 0, 0, thus no error should occur.
		log.Error().Err(err).Msgf("SetView error: %s", err)
	}
}

// viewLines returns the lines to render on the screen
func (v *View) viewLines() [][]cell {
	if !v.Wrap {
		return v.lines
	}

	renderLines := [][]cell{}
	for _, viewLine := range v.lines {
		for {
			lineToRender, _, end := v.takeLine(&viewLine)
			renderLines = append(renderLines, lineToRender)
			if end {
				break
			}
		}
	}
	return renderLines
}

// IsTainted tells us if the view is tainted
func (v *View) IsTainted() bool {
	return v.tainted
}

// draw re-draws the view's contents.
func (v *View) draw() error {
	if !v.Visible {
		return nil
	}

	maxX, maxY := v.Size()

	if v.Wrap {
		if maxX == 0 {
			// Just return here, there is no need to try drawing chars in a too small frame
			// Nor is it needed to return an error, there is just no space
			return nil
		}
		v.ox = 0
	}

	if !v.tainted && v.contentCache != nil {
		for _, cell := range v.contentCache {
			if err := v.setRune(cell.x, cell.y, cell.chr, cell.fgColor, cell.bgColor); err != nil {
				return err
			}
		}
		return nil
	}

	linesToRender := v.viewLines()

	if v.Autoscroll && len(linesToRender) > maxY {
		v.oy = len(linesToRender) - maxY - 1
	}

	newCache := []cellCache{}
	y := 0
	for lineIndex, line := range linesToRender {
		if lineIndex < v.oy {
			continue
		}
		if y >= maxY {
			break // No need to render out of screen chars
		}

		x := 0
		for charIndex, char := range line {
			if charIndex < v.ox {
				continue
			}
			if x >= maxX {
				break // No need to render out of screen chars
			}

			fgColor := char.fgColor
			if fgColor == ColorDefault {
				fgColor = v.FgColor
			}
			bgColor := char.bgColor
			if bgColor == ColorDefault {
				bgColor = v.BgColor
			}

			newCache = append(newCache, cellCache{
				chr:     char.chr,
				bgColor: bgColor,
				fgColor: fgColor,
				x:       x,
				y:       y,
			})
			if err := v.setRune(x, y, char.chr, fgColor, bgColor); err != nil {
				return err
			}
			if char.chr == 0 {
				x++ // if NULL increase, so `SetWritePos` can be used (NULL translate to SPACE in setRune)
			} else {
				x += runewidth.RuneWidth(char.chr)
			}
		}
		y++
	}

	v.contentCache = newCache
	v.tainted = false //A.N.
	return nil
}

func (v *View) ClearCache() {
	v.contentCache = nil
}

// Clear empties the view and resets the view offsets, cursor position, read offsets and write offsets
func (v *View) Clear() {
	v.writeMutex.Lock()
	defer v.writeMutex.Unlock()
	v.Rewind()
	v.tainted = true
	v.ei.reset()
	v.lines = [][]cell{}
	v.hotspots = []*Hotspot{}
	v.activeHotspot = nil
	v.SetCursor(0, 0)
	v.SetOrigin(0, 0)
	v.clearRunes()
}

// linesPosOnScreen returns based on the view lines the x and y location
// the viewX and viewY are NOT based on the view offsets
// isOnScreen is false if the selected corodinates is not on screen (this is based on the view offsets)
func (v *View) linesPosOnScreen(x, y int) (viewX int, viewY int, visable bool) {
	if x < 0 || y < 0 {
		return
	}

	maxX, maxY := v.Size()
	if !v.Wrap {
		viewX = x
		viewY = y
		visable = viewY >= v.oy && viewY < v.oy+maxY && viewX >= v.ox && viewX < v.ox+maxX
		return
	}

	var line []cell
	found := false

	for lineIndex, viewLine := range v.lines {
		if lineIndex == y {
			line = viewLine
			found = true
			break
		}

		for {
			_, _, end := v.takeLine(&viewLine)
			viewY++
			if end {
				break
			}
		}
	}

	if found {
		for {
			lineChars, width, end := v.takeLine(&line)
			lenLineChars := len(lineChars)
			if x < lenLineChars {
				x = lineWidth(lineChars[:x])
				break
			} else {
				x -= lenLineChars
			}

			if end {
				x += width
				break
			}
			viewY++
		}
	} else {
		if y < len(v.lines) {
			viewY = y
		} else {
			viewY += y - len(v.lines)
		}
	}

	viewY += x / maxX
	viewX = x - ((x / maxX) * maxX)

	visable = viewY >= v.oy && viewY < v.oy+maxY && viewX >= v.ox && viewX < v.ox+maxX
	return
}

// clearRunes erases all the cells in the view.
func (v *View) clearRunes() {
	maxX, maxY := v.Size()
	for x := 0; x < maxX; x++ {
		for y := 0; y < maxY; y++ {
			tcellSetCell(v.x0+x+1, v.y0+y+1, ' ', v.FgColor, v.BgColor, v.outMode)
		}
	}
}

// BufferLines returns the lines in the view's internal
// buffer.
func (v *View) BufferLines() []string {
	lines := make([]string, len(v.lines))
	for i, l := range v.lines {
		str := lineType(l).String()
		str = strings.Replace(str, "\x00", " ", -1)
		lines[i] = str
	}
	return lines
}

// Buffer returns a string with the contents of the view's internal
// buffer.
func (v *View) Buffer() string {
	return linesToString(v.lines)
}

// ViewBufferLines returns the lines in the view's internal
// buffer that is shown to the user.
func (v *View) ViewBufferLines() []string {
	viewLines := v.viewLines()
	lines := make([]string, len(viewLines))
	for i, line := range viewLines {
		str := lineType(line).String()
		str = strings.Replace(str, "\x00", " ", -1)
		lines[i] = str
	}
	return lines
}

// LinesHeight is the count of view lines (i.e. lines excluding wrapping)
func (v *View) LinesHeight() int {
	return len(v.lines)
}

// ViewLinesHeight is the count of view lines (i.e. lines including wrapping)
func (v *View) ViewLinesHeight() int {
	if !v.tainted && v.contentCache != nil && len(v.contentCache) > 0 {
		// Use the cache if availabe, it's just a bit faster than re-calculating all frame cells
		return v.contentCache[len(v.contentCache)-1].y + 1
	}
	return len(v.viewLines())
}

// ViewBuffer returns a string with the contents of the view's buffer that is
// shown to the user.
func (v *View) ViewBuffer() string {
	return linesToString(v.viewLines())
}

// Line returns a string with the line of the view's internal buffer
// at the position corresponding to the point (x, y).
func (v *View) Line(y int) (string, error) {
	if y < 0 || y >= len(v.lines) {
		return "", ErrInvalidPoint
	}

	return lineType(v.lines[y]).String(), nil
}

// Word returns a string with the word of the view's internal buffer
// at the position corresponding to the point (x, y).
func (v *View) Word(x, y int) (string, error) {
	if x < 0 || y < 0 || y >= len(v.lines) || x >= len(v.lines[y]) {
		return "", ErrInvalidPoint
	}

	str := lineType(v.lines[y]).String()

	nl := strings.LastIndexFunc(str[:x], indexFunc)
	if nl == -1 {
		nl = 0
	} else {
		nl = nl + 1
	}
	nr := strings.IndexFunc(str[x:], indexFunc)
	if nr == -1 {
		nr = len(str)
	} else {
		nr = nr + x
	}
	return string(str[nl:nr]), nil
}

// indexFunc allows to split lines by words taking into account spaces
// and 0.
func indexFunc(r rune) bool {
	return r == ' ' || r == 0
}

// SetLine changes the contents of an existing line.
func (v *View) SetLine(y int, text string) error {
	if y < 0 || y >= len(v.lines) {
		err := ErrInvalidPoint
		return err
	}

	v.tainted = true
	line := make([]cell, 0)
	for _, r := range text {
		c := v.parseInput(r)
		line = append(line, c...)
	}
	v.lines[y] = line
	return nil
}

// SetHighlight toggles highlighting of separate lines, for custom lists
// or multiple selection in views.
func (v *View) SetHighlight(y int, on bool) error {
	if y < 0 || y >= len(v.lines) {
		err := ErrInvalidPoint
		return err
	}

	line := v.lines[y]
	cells := make([]cell, 0)
	for _, c := range line {
		if on {
			c.bgColor = v.SelBgColor
			c.fgColor = v.SelFgColor
		} else {
			c.bgColor = v.BgColor
			c.fgColor = v.FgColor
		}
		cells = append(cells, c)
	}
	v.tainted = true
	v.lines[y] = cells
	return nil
}

func lineWidth(line []cell) (n int) {
	for i := range line {
		if line[i].chr == 0 {
			n++ // if it's NULL character, it's translated to SPACE in setRune
		} else {
			n += runewidth.RuneWidth(line[i].chr)
		}
	}

	return
}

// takeLine slices one visable line from l and returns the sliced part
func (v *View) takeLine(l *[]cell) (visableLine []cell, width int, end bool) {
	if l == nil {
		panic("take line l can't be nil")
	}

	visableLine = []cell{}

	if len(*l) == 0 {
		end = true
		width = 0
		return
	}

	maxX, _ := v.Size()
	i := 0
	cell := cell{}

	for i, cell = range *l {
		chr := cell.chr
		charWidth := 1 // default for NULL character (translated to SPACE in setRune)
		if chr != 0 {
			charWidth = runewidth.RuneWidth(chr)
		}

		if width+charWidth > maxX {
			i-- // decrease as this character is not included
			break
		}

		width += charWidth
		visableLine = append(visableLine, cell)
		if width == maxX {
			break
		}
	}

	i++
	end = i == len(*l)
	*l = (*l)[i:]

	return
}

func linesToString(lines [][]cell) string {
	str := make([]string, len(lines))
	for i := range lines {
		rns := make([]rune, 0, len(lines[i]))
		line := lineType(lines[i]).String()
		for _, c := range line {
			if c == '\x00' {
				rns = append(rns, ' ')
			} else {
				rns = append(rns, c)
			}
		}
		str[i] = string(rns)
	}

	return strings.Join(str, "\n")
}

func (v *View) ScrollUp(n int) {
	if v.ScrollBar {
		v.oy -= n
		if v.oy < 0 {
			v.oy = 0
		}
		v.SetOrigin(v.ox, v.oy)
	}
}

func (v *View) ScrollDown(n int) {
	if v.ScrollBar {
		height := v.y1 - v.y0 - 1
		lines := len(v.lines)

		if lines > height {
			v.oy += n
			if lines-v.oy < height {
				v.oy = lines - height
			}
		}
	}
}

func (v *View) ScrollTop() {
	v.oy = 0
	v.SetOrigin(v.ox, v.oy)
}

func (v *View) ScrollBottom() {
	height := v.y1 - v.y0 - 1
	lines := len(v.lines)

	if lines > height {
		v.oy = lines - height
	}
}

func (v *View) MouseOverScrollbar() bool {

	if v.gui.popup != nil && v.gui.popup.View != nil { // ignore outside events
		if !strings.HasPrefix(v.name, v.gui.popup.View.name+".") {
			return false
		}
	}

	return v.ScrollBar &&
		v.ScrollBarStatus.shown &&
		v.gui.mouseX == v.x1-1 &&
		v.gui.mouseY >= v.y0+v.ScrollBarStatus.position &&
		v.gui.mouseY < v.y0+v.ScrollBarStatus.position+v.ScrollBarStatus.height
}

func ParseTag(tag string) (string, map[string]string) {

	// Regular expression to match the whole tag, capturing the tag name and parameters
	tagRe := regexp.MustCompile(REGEX_TAGS)

	tagName := ""
	tagParams := make(map[string]string)

	tagMatch := tagRe.FindStringSubmatch(tag)
	if len(tagMatch) > 0 {
		tagName = tagMatch[1]
		params := tagMatch[2]
		if params != "" {
			// Regular expression to match individual parameters
			paramRe := regexp.MustCompile(REGEX_SINGLE_TAG)
			paramMatches := paramRe.FindAllStringSubmatch(params, -1)

			for _, paramMatch := range paramMatches {
				paramName := paramMatch[1]
				paramValue := "true" // Default value for flag-like parameters

				if len(paramMatch) > 2 && paramMatch[2] != "" {
					paramValue = paramMatch[2]
					if (strings.HasPrefix(paramValue, `"`) && strings.HasSuffix(paramValue, `"`)) ||
						(strings.HasPrefix(paramValue, `'`) && strings.HasSuffix(paramValue, `'`)) {
						// Remove quotes from the value
						paramValue = paramValue[1 : len(paramValue)-1]
					}
				}

				tagParams[paramName] = paramValue

			}
		}
	}

	return tagName, tagParams
}

func (v *View) AddTag(text string) error {
	tagName, tagParams := ParseTag(text)
	if err := v.AddTagEx(tagName, tagParams); err != nil {
		return err
	}

	return nil
}

func (v *View) AddTagEx(tagName string, tagParams map[string]string) error {
	switch tagName {
	case "l": // link
		v.AddLink(tagParams["text"], tagParams["action"], tagParams["tip"], tagParams["id"])
	case "button": // button
		if tagParams["id"] == "" {
			tagParams["id"] = tagParams["text"]
		}
		v.AddButton(tagParams["text"], "button "+tagParams["id"], tagParams["id"], tagParams["tip"], tagParams["color"], tagParams["bgcolor"])
	case "input": // input
		v.AddInput(tagParams)
	case "t": // text input
		v.AddTextInput(tagParams)
	case "select": // combo box
		v.AddSelect(tagParams)
	}
	return nil
}

func (v *View) GetHotspotById(id string) *Hotspot {
	for _, hs := range v.hotspots {
		if hs.ID == id {
			return hs
		}
	}
	return nil
}

func GetTagLength(tagName string, tagParams map[string]string) int {
	switch tagName {
	case "l": // link
		return utf8.RuneCountInString(tagParams["text"])
	case "button": // button
		return utf8.RuneCountInString(tagParams["text"]) + 2
	case "input": // input
		size, _ := strconv.Atoi(tagParams["size"])
		return size
	case "t": // text input
		width, _ := strconv.Atoi(tagParams["width"])
		return width
	case "select": // droplist
		size, _ := strconv.Atoi(tagParams["size"])
		return size
	}
	return 0
}

func AddCells(cells []cell, fg, bg Attribute, text string) []cell {
	if cells == nil {
		cells = []cell{}
	}

	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		cells = append(cells, cell{runes[i], bg, fg})
	}
	return cells
}

func (v *View) AddLink(text, value, tip, id string) error {
	cells := AddCells(nil, v.EmFgColor, v.BgColor, text)
	cells_highligted := AddCells(nil, v.SelFgColor, v.SelBgColor, text)

	c := PopoupControl{
		Type: C_LINK,
		x0:   v.wx,
		y0:   v.wy,
		x1:   v.wx + len(cells),
		y1:   v.wy + 1,
	}

	hs, err := v.AddHotspot(v.wx, v.wy, id, value, tip, cells, cells_highligted)

	if err == nil {
		v.Write([]byte(text))
		c.Hotspot = hs
		v.Controls = append(v.Controls, &c)
	}

	return err
}

func (v *View) ParseColor(color string) Attribute {
	c := strings.TrimSpace(color)

	switch c {
	case "":
		return v.FgColor
	case "v.BgColor":
		return v.BgColor
	case "v.FgColor":
		return v.FgColor
	case "v.SelBgColor":
		return v.SelBgColor
	case "v.SelFgColor":
		return v.SelFgColor
	case "v.SubTitleFgColor":
		return v.SubTitleFgColor
	case "v.SubTitleBgColor":
		return v.SubTitleBgColor
	case "v.EmFgColor":
		return v.EmFgColor
	case "v.TitleAttrib":
		return v.TitleAttrib
	case "v.SubTitleAttrib":
		return v.SubTitleAttrib
	case "g.BgColor":
		return v.gui.BgColor
	case "g.FgColor":
		return v.gui.FgColor
	case "g.SelBgColor":
		return v.gui.SelBgColor
	case "g.SelFgColor":
		return v.gui.SelFgColor
	case "g.SubTitleFgColor":
		return v.gui.SubTitleFgColor
	case "g.SubTitleBgColor":
		return v.gui.SubTitleBgColor
	case "g.EmFgColor":
		return v.gui.EmFgColor
	case "g.TitleAttrib":
		return v.gui.FrameColor
	case "g.InputBgColor":
		return v.gui.InputBgColor
	case "g.ErrorFgColor":
		return v.gui.ErrorFgColor
	case "g.ActionBgColor":
		return v.gui.ActionBgColor
	case "g.ActionFgColor":
		return v.gui.ActionFgColor
	case "g.ActionSelBgColor":
		return v.gui.ActionSelBgColor
	case "g.ActionSelFgColor":
		return v.gui.ActionSelFgColor
	case "g.HelpBgColor":
		return v.gui.HelpBgColor
	case "g.HelpFgColor":
		return v.gui.HelpFgColor
	}

	return GetColor(color)
}

func (v *View) AddButton(text, value, tip, id, color, bgcolor string) error {
	fg, bg := v.BgColor, v.EmFgColor

	if color != "" {
		fg = v.ParseColor(color)
	}

	if bgcolor != "" {
		bg = v.ParseColor(bgcolor)
	}

	cells := AddCells(nil, bg, v.BgColor, "\ue0b6")
	cells = AddCells(cells, fg|AttrBold, bg, text)
	cells = AddCells(cells, bg, v.BgColor, "\ue0b4")

	cells_highligted := AddCells(nil, v.SelFgColor, v.SelBgColor, "\ue0b6")
	cells_highligted = AddCells(cells_highligted, v.SelBgColor|AttrBold, v.SelFgColor, text)
	cells_highligted = AddCells(cells_highligted, v.SelFgColor, v.SelBgColor, "\ue0b4")

	if value == "" {
		value = "button " + text
	}

	if tip == "" {
		tip = text
	}

	c := PopoupControl{
		Type: C_BUTTON,
		ID:   value,
		x0:   v.wx,
		y0:   v.wy,
		x1:   v.wx + len(cells),
		y1:   v.wy + 1,
	}

	hs, err := v.AddHotspot(v.wx, v.wy, id, value, tip, cells, cells_highligted)

	if err == nil {
		v.writeMutex.Lock()
		defer v.writeMutex.Unlock()
		v.makeWriteable(v.wx, v.wy)
		v.writeCells(v.wx, v.wy, cells)
		v.wx += len(cells)
		c.Hotspot = hs
		v.Controls = append(v.Controls, &c)
	}
	return err
}

func (v *View) AddSelect(tagParams map[string]string) error {
	size, _ := strconv.Atoi(tagParams["size"])
	if size == 0 {
		return errors.New("input tag must have a size attribute")
	}

	value := tagParams["value"]
	text := value

	if utf8.RuneCountInString(text) > size-1 {
		text = text[:size-1]
	}

	if utf8.RuneCountInString(text) < size-1 {
		text += strings.Repeat(" ", size-1-len(text))
	}

	cells := AddCells(nil, v.FgColor, v.gui.InputBgColor, text)
	cells = AddCells(cells, v.EmFgColor, v.gui.InputBgColor, ICON_DROPLIST)
	cells_highligted := AddCells(nil, v.SelFgColor, v.gui.InputBgColor, text+ICON_DROPLIST)

	list := tagParams["list"]

	c := PopoupControl{
		Type:  C_SELECT,
		ID:    tagParams["id"],
		Value: value,
		x0:    v.wx,
		y0:    v.wy,
		x1:    v.wx + len(cells),
		y1:    v.wy + 1,
		Items: strings.Split(list, ","),
	}

	hs, err := v.AddHotspot(v.wx, v.wy, tagParams["id"], "droplist "+tagParams["id"], "", cells, cells_highligted)

	if err == nil {
		v.writeMutex.Lock()
		v.makeWriteable(v.wx, v.wy)
		defer v.writeMutex.Unlock()
		v.writeCells(v.wx, v.wy, cells)
		v.wx += len(cells)
		c.Hotspot = hs
		v.Controls = append(v.Controls, &c)
	}
	return err
}

func (v *View) AddInput(tagParams map[string]string) error {
	name := v.name + "." + tagParams["id"]
	if _, err := v.gui.View(name); err == nil {
		log.Error().Msg("input with id " + name + " already exists")
		return errors.New("input with id " + name + " already exists")
	}

	size, _ := strconv.Atoi(tagParams["size"])
	if size == 0 {
		log.Error().Msg("input tag must have a size attribute")
		return errors.New("input tag must have a size attribute")
	}

	c := PopoupControl{
		Type: C_INPUT,
		ID:   tagParams["id"],
		x0:   v.wx,
		y0:   v.wy,
		x1:   v.wx + size + 1,
		y1:   v.wy + 2,
	}

	if v, err := v.gui.SetView(name, v.x0+v.wx, v.y0+v.wy, v.x0+v.wx+size+1, v.y0+v.wy+2, 0); err != nil {
		if !errors.Is(err, ErrUnknownView) {
			log.Error().Err(err).Msg("SetView error")
			return err
		}
		v.Frame = false

		if tagParams["masked"] == "true" {
			v.Mask = '*'
		}

		v.BgColor = v.gui.InputBgColor
		v.Editor = EditorFunc(PopupNavigation)

		v.Write([]byte(tagParams["value"]))

		v.Editable = true
		v.Wrap = false
		v.Autoscroll = true
		c.View = v
	}

	v.Controls = append(v.Controls, &c)

	// write placeholder
	v.Write([]byte(strings.Repeat(" ", size)))

	return nil
}

func (v *View) AddTextInput(tagParams map[string]string) error {
	name := v.name + "." + tagParams["id"]
	if _, err := v.gui.View(name); err == nil {
		return errors.New("input with id " + name + " already exists")
	}

	width, _ := strconv.Atoi(tagParams["width"])
	if width == 0 {
		return errors.New("input tag must have a width attribute")
	}

	height, _ := strconv.Atoi(tagParams["height"])
	if height == 0 {
		return errors.New("input tag must have a height attribute")
	}

	c := PopoupControl{
		Type: C_TEXT_INPUT,
		ID:   tagParams["id"],
		x0:   v.wx,
		y0:   v.wy,
		x1:   v.wx + width + 1,
		y1:   v.wy + height + 1,
	}

	if v, err := v.gui.SetView(name, v.x0+v.wx, v.y0+v.wy, v.x0+v.wx+width+1, v.y0+v.wy+height+1, 0); err != nil {
		if !errors.Is(err, ErrUnknownView) {
			return err
		}
		v.Frame = false

		if tagParams["masked"] == "true" {
			v.Mask = '*'
		}

		v.BgColor = v.gui.InputBgColor
		v.Editor = EditorFunc(PopupNavigation)

		v.Write([]byte(tagParams["value"]))

		v.Editable = true
		v.Wrap = true
		v.Autoscroll = true
		c.View = v
	}

	v.Controls = append(v.Controls, &c)

	// write placeholder
	v.Write([]byte(strings.Repeat(" ", width-1)))

	return nil
}

func PopupNavigation(v *View, key Key, ch rune, mod Modifier) {

	if v.gui.popup == nil || v.gui.popup.View == nil {
		return
	}
	switch key {
	case KeyEsc:
		v.gui.popup.gui.HidePopup()
	case KeyEnter:
		if v.gui.popup.View.ControlInFocus != -1 {
			c := v.gui.popup.View.Controls[v.gui.popup.View.ControlInFocus]
			switch c.Type {
			case C_LINK:
				v.gui.popup.View.OnClickHotspot(v.gui.popup.View, c.Hotspot)
			case C_BUTTON:
				v.gui.popup.View.OnClickHotspot(v.gui.popup.View, c.Hotspot)
			case C_INPUT:
				v.gui.popup.View.FocusNext()
			case C_SELECT:
				v.gui.popup.View.ShowDropList(c)
			case C_TEXT_INPUT:
				DefaultEditor.Edit(v, key, ch, mod)
			}
		}
	case KeySpace:
		if v.gui.popup.View.ControlInFocus != -1 {
			c := v.gui.popup.View.Controls[v.gui.popup.View.ControlInFocus]
			switch c.Type {
			case C_LINK:
				v.gui.popup.View.OnClickHotspot(v.gui.popup.View, c.Hotspot)
			case C_BUTTON:
				v.gui.popup.View.OnClickHotspot(v.gui.popup.View, c.Hotspot)
			default:
				DefaultEditor.Edit(v, key, ch, mod)
			}
		}
	case KeyTab:
		v.gui.popup.View.FocusNext()
	case KeyBacktab:
		v.gui.popup.View.FocusPrev()
	case KeyArrowRight:
		if v.ControlInFocus != -1 &&
			v.Controls[v.ControlInFocus].Type != C_INPUT {
			v.FocusNext()
		} else {
			DefaultEditor.Edit(v, key, ch, mod)
		}
	case KeyArrowLeft:
		if v.ControlInFocus != -1 &&
			v.Controls[v.ControlInFocus].Type != C_INPUT {
			v.gui.popup.View.FocusPrev()
		} else {
			DefaultEditor.Edit(v, key, ch, mod)
		}
	case KeyArrowUp:
		if v.gui.popup.View.ControlInFocus != -1 &&
			v.gui.popup.View.Controls[v.gui.popup.View.ControlInFocus].Type != C_TEXT_INPUT {
			v.gui.popup.View.FocusPrev()
		} else {
			DefaultEditor.Edit(v, key, ch, mod)
		}
	case KeyArrowDown:
		if v.gui.popup.View.ControlInFocus != -1 &&
			v.gui.popup.View.Controls[v.gui.popup.View.ControlInFocus].Type != C_TEXT_INPUT {
			v.gui.popup.View.FocusNext()
		} else {
			DefaultEditor.Edit(v, key, ch, mod)
		}
	default:
		if v.gui.popup.View.ControlInFocus != -1 {
			c := v.gui.popup.View.Controls[v.gui.popup.View.ControlInFocus]
			switch c.Type {
			case C_INPUT, C_TEXT_INPUT:
				DefaultEditor.Edit(v, key, ch, mod)
				if v.gui.popup.OnChange != nil {
					v.gui.popup.OnChange(v.gui.popup, c)
				}
			}
		}
	}
}

func (v *View) SetFocus(i int) {

	if v.DropList != nil {
		v.gui.DeleteView(v.DropList.name)
		v.DropList = nil
	}

	L := len(v.Controls)
	if L == 0 {
		return
	}

	i = ((i % L) + L) % L

	v.ControlInFocus = i % len(v.Controls)

	switch v.Controls[v.ControlInFocus].Type {
	case C_INPUT, C_TEXT_INPUT:
		v.gui.SetCurrentView(v.Controls[i].View.name)
	default:
		v.gui.currentView = nil
		screen.HideCursor()
	}
}

func (v *View) FocusNext() {
	v.SetFocus(v.ControlInFocus + 1)
}

func (v *View) FocusPrev() {
	v.SetFocus(v.ControlInFocus - 1)
}

func (v *View) GetInput(id string) string {
	for _, c := range v.Controls {
		if c.ID == id {
			switch c.Type {
			case C_INPUT, C_TEXT_INPUT:
				if c.View != nil {
					return c.View.Buffer()
				}
			case C_SELECT:
				return c.Value
			}
		}
	}
	return ""
}

func (v *View) SetInput(id, value string) {
	for _, c := range v.Controls {
		if c.ID == id {
			switch c.Type {
			case C_INPUT, C_TEXT_INPUT:
				c.View.Clear()
				c.View.Write([]byte(value))
			case C_SELECT:
				c.Value = value
				text := value
				size := c.x1 - c.x0

				if len(text) > size-1 {
					text = text[:size-1]
				}

				if len(text) < size-1 {
					text += strings.Repeat(" ", size-1-len(text))
				}

				c.Cells = AddCells(nil, v.FgColor, v.gui.InputBgColor, text)
				c.Cells = AddCells(c.Cells, v.EmFgColor, v.gui.InputBgColor, ICON_DROPLIST)
				c.CellsHighligted = AddCells(nil, v.SelFgColor, v.gui.InputBgColor, text+ICON_DROPLIST)
			}
		}
	}
}

func (v *View) SetList(id string, list []string) {
	for _, c := range v.Controls {
		if c.Type == C_SELECT && c.ID == id {
			c.Items = list
			break
		}
	}
}

func EstimateTemplateLines(template string, width int) int {
	n_lines := 0

	if width < 3 {
		return 0 // no space to render
	}

	lines := strings.Split(template, "\n")

	if len(lines) == 0 {
		return 0
	}

	autowrap := false

	for _, line := range lines {
		splitted_lines := []string{}

		if autowrap {
			spaces := []int{}
			for calcLineWidth(line) > width && len(line) > 0 {
				in_tag := false
				for i, r := range line {
					switch r {
					case '<':
						in_tag = true
					case '>':
						in_tag = false
					case ' ':
						if !in_tag {
							spaces = append(spaces, i)
						}
					}
				}

				splited := false
				for i := len(spaces) - 2; i > 0; i-- {
					try := line[:spaces[i]]

					if calcLineWidth(try) <= width {
						splitted_lines = append(splitted_lines, try)
						line = line[spaces[i]+1:]
						splited = true
						break
					}
				}

				if !splited {
					break
				}
			}
		}

		splitted_lines = append(splitted_lines, line)
		n_lines += len(splitted_lines)
	}

	return n_lines + 1
}

func (v *View) RenderTemplate(template string) error {
	v.Clear()

	width := v.x1 - v.x0 - 1

	if width < 3 {
		return nil // no space to render
	}

	re := regexp.MustCompile(REGEX_TAGS)
	lines := strings.Split(template, "\n")

	if len(lines) == 0 {
		return errors.New("empty template")
	}

	centered := false
	bold := false
	blink := false
	reverse := false
	underline := false
	dim := false
	italic := false
	strikethrough := false
	var FGColor, BGColor *Attribute

	autowrap := false

	for _, line := range lines {

		if strings.Contains(line, "\t") {
			log.Warn().Msgf("tabs are not allowed in templates : %s", line)
			line = strings.ReplaceAll(line, "\t", " ")
		}

		splitted_lines := []string{}

		if autowrap {
			spaces := []int{}
			for calcLineWidth(line) > width && len(line) > 0 {
				in_tag := false
				for i, r := range line {
					switch r {
					case '<':
						in_tag = true
					case '>':
						in_tag = false
					case ' ':
						if !in_tag {
							spaces = append(spaces, i)
						}
					}
				}

				splited := false
				for i := len(spaces) - 2; i > 0; i-- {
					try := line[:spaces[i]]

					if calcLineWidth(try) <= width {
						splitted_lines = append(splitted_lines, try)
						line = line[spaces[i]+1:]
						splited = true
						break
					}
				}

				if !splited {
					break
				}
			}
		}

		splitted_lines = append(splitted_lines, line)

		var cells []cell
		for _, l := range splitted_lines {
			left := 0
			matches := re.FindAllStringIndex(l, -1)
			draw_line := false
			draw_line_text := ""
			for _, match := range matches {

				if v.wx == 0 && centered {
					n := (width - calcLineWidth(l)) / 2
					for i := 0; i < n; i++ {
						v.Write([]byte(" "))
					}
				}

				color := v.FgColor
				if FGColor != nil {
					color = *FGColor
				}

				if bold {
					color |= AttrBold
				}
				if underline {
					color |= AttrUnderline
				}
				if reverse {
					color |= AttrReverse
				}
				if dim {
					color |= AttrDim
				}
				if italic {
					color |= AttrItalic
				}
				if strikethrough {
					color |= AttrStrikeThrough
				}
				if blink {
					color |= AttrBlink
				}
				bgcolor := v.BgColor
				if BGColor != nil {
					bgcolor = *BGColor
				}
				cells = AddCells(nil, color, bgcolor, l[left:match[0]])

				v.makeWriteable(v.wx, v.wy)
				v.writeCells(v.wx, v.wy, cells)
				v.wx += len(cells)

				tag := l[match[0]:match[1]]
				tagName, tagParams := ParseTag(tag)

				switch tagName {
				case "c":
					centered = true
				case "/c":
					centered = false
				case "w":
					autowrap = true
				case "/w":
					autowrap = false
				case "b":
					bold = true
				case "/b":
					bold = false
				case "u":
					underline = true
				case "/u":
					underline = false
				case "r":
					reverse = true
				case "/r":
					reverse = false
				case "dim":
					dim = true
				case "/dim":
					dim = false
				case "i":
					italic = true
				case "/i":
					italic = false
				case "s":
					strikethrough = true
				case "/s":
					strikethrough = false
				case "blink":
					blink = true
				case "/blink":
					blink = false
				case "color":
					if tagParams["fg"] != "" {
						c := v.ParseColor(tagParams["fg"])
						FGColor = &c
					}
					if tagParams["bg"] != "" {
						c := v.ParseColor(tagParams["bg"])
						BGColor = &c
					}

				case "/color":
					FGColor = nil
					BGColor = nil
				case "line":
					draw_line = true
					draw_line_text = tagParams["text"]
				default:
					v.AddTag(tag)
				}

				left = match[1]
			}

			if v.wx == 0 && centered {
				n := (width - calcLineWidth(l)) / 2
				for i := 0; i < n; i++ {
					v.Write([]byte(" "))
				}
			}

			color := v.FgColor
			if FGColor != nil {
				color = *FGColor
			}

			if bold {
				color |= AttrBold
			}
			if underline {
				color |= AttrUnderline
			}
			if reverse {
				color |= AttrReverse
			}
			if dim {
				color |= AttrDim
			}
			if italic {
				color |= AttrItalic
			}
			if strikethrough {
				color |= AttrStrikeThrough
			}
			if blink {
				color |= AttrBlink
			}

			if draw_line {
				v.wx = 0
				left = 0
				if draw_line_text == "" {
					l = strings.Repeat("━", width)
				} else {
					tl := utf8.RuneCountInString(draw_line_text)
					if tl >= width-4 {
						l = draw_line_text[:width]
					} else {
						if (width-tl-2)/2 > 0 {
							l = strings.Repeat("━", (width-tl-2)/2)
						}
						l = l + " " + draw_line_text + " "
						left := width - utf8.RuneCountInString(l)
						if left > 0 {
							l = l + strings.Repeat("━", left)
						}
					}
				}
			}

			bgcolor := v.BgColor
			if BGColor != nil {
				bgcolor = *BGColor
			}

			cells = AddCells(nil, color, bgcolor, l[left:])
			v.makeWriteable(v.wx, v.wy)
			v.writeCells(v.wx, v.wy, cells)
			v.wx += len(cells)

			v.Write([]byte("\n"))
		}

	}

	return nil
}

func calcLineWidth(line string) int {
	l := utf8.RuneCountInString(line)
	re := regexp.MustCompile(REGEX_TAGS)
	matches := re.FindAllStringIndex(line, -1)
	for _, match := range matches {
		tag := line[match[0]:match[1]]
		tagName, tagParams := ParseTag(tag)
		l = l - utf8.RuneCountInString(tag) + GetTagLength(tagName, tagParams)
	}

	return l
}
