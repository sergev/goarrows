package ui

import (
	"github.com/gdamore/tcell/v2"
	"goarrows/game"
)

// GridSize returns terminal width and height in cells for a w×h board (one char per cell).
func GridSize(w, h int) (gw, gh int) {
	return w, h
}

// DrawGrid paints the board at (ox, oy) with no cell borders—one rune per logical cell.
func DrawGrid(s tcell.Screen, ox, oy int, b game.Board, cursorX, cursorY int, base, cursor tcell.Style) {
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			st := base
			if x == cursorX && y == cursorY {
				st = cursor
			}
			r := DisplayRune(b.At(x, y))
			s.SetContent(ox+x, oy+y, r, nil, st)
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
