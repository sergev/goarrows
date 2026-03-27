package ui

import (
	"github.com/gdamore/tcell/v2"
	"goarrows/game"
)

// GridSize returns terminal width and height in cells for a w×h board (with box lines).
func GridSize(w, h int) (gw, gh int) {
	return 2*w + 1, 2*h + 1
}

// DrawGrid paints the bordered grid at (ox, oy) using s.SetContent.
// cursorCell is highlighted with cursorStyle when in bounds; use (-1,-1) to skip.
func DrawGrid(s tcell.Screen, ox, oy int, b game.Board, cursorX, cursorY int, base, cursor tcell.Style) {
	w, h := b.W, b.H
	for j := 0; j < 2*h+1; j++ {
		for i := 0; i < 2*w+1; i++ {
			x, y := ox+i, oy+j
			st := base
			cx, cy := (i-1)/2, (j-1)/2
			if i%2 == 1 && j%2 == 1 && cursorX == cx && cursorY == cy {
				st = cursor
			}
			r := gridRune(b, w, h, i, j)
			s.SetContent(x, y, r, nil, st)
		}
	}
}

func gridRune(b game.Board, w, h, i, j int) rune {
	switch {
	case j == 0 && i == 0:
		return '┌'
	case j == 0 && i == 2*w:
		return '┐'
	case j == 2*h && i == 0:
		return '└'
	case j == 2*h && i == 2*w:
		return '┘'
	case j%2 == 1 && i%2 == 1:
		c := b.At(i/2, j/2)
		return ArrowRune(c)
	case j%2 == 1 && i%2 == 0:
		return '│'
	case j == 0:
		if i%2 == 1 {
			return '─'
		}
		return '┬'
	case j == 2*h:
		if i%2 == 1 {
			return '─'
		}
		return '┴'
	case i == 0:
		return '├'
	case i == 2*w:
		return '┤'
	case i%2 == 1:
		return '─'
	default:
		return '┼'
	}
}

// ArrowRune maps a cell to ◀▶▲▼ or space.
func ArrowRune(c game.Cell) rune {
	if c.Empty {
		return ' '
	}
	switch c.Dir {
	case game.North:
		return '▲'
	case game.South:
		return '▼'
	case game.West:
		return '◀'
	case game.East:
		return '▶'
	default:
		return '?'
	}
}
