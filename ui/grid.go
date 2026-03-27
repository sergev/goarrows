package ui

import (
	"github.com/gdamore/tcell/v2"
	"goarrows/game"
)

// GridSize returns terminal width and height for a w×h logical board: each
// logical column uses two screen columns (glyph at 2x, optional ─ bridge at 2x+1),
// so width is 2*w-1 and height is h.
func GridSize(w, h int) (gw, gh int) {
	if w <= 0 {
		return 0, h
	}
	return 2*w - 1, h
}

// DrawGrid paints the board at (ox, oy). Logical cell (x,y) is drawn at screen
// (ox+2*x, oy+y). Between (x,y) and (x+1,y), a '─' is drawn when the path links
// horizontally so lines stay visually continuous.
func DrawGrid(s tcell.Screen, ox, oy int, b game.Board, cursorX, cursorY int, base, cursor tcell.Style) {
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			st := base
			if x == cursorX && y == cursorY {
				st = cursor
			}
			r := DisplayRune(b.At(x, y))
			s.SetContent(ox+2*x, oy+y, r, nil, st)
		}
		for x := 0; x+1 < b.W; x++ {
			st := base
			if y == cursorY && (x == cursorX || x+1 == cursorX) {
				st = cursor
			}
			ch := ' '
			if game.HorizontalLink(b, x, y) {
				ch = '─'
			}
			s.SetContent(ox+2*x+1, oy+y, ch, nil, st)
		}
	}
}

// DisplayRune returns the glyph to draw for a cell (space if empty).
func DisplayRune(c game.Cell) rune {
	if c.IsEmpty() {
		return ' '
	}
	return c.R
}
