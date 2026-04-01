package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// ModalOverlay is a centered modal with prebuilt text lines and tone.
type ModalOverlay struct {
	Positive bool
	Lines    []string
}

// FormatLives formats the current life count for HUD display.
func FormatLives(current, start int) string {
	if start < 0 {
		return "∞"
	}
	return fmt.Sprintf("%d", current)
}

// DrawGeneratingOverlay paints a centered "creating level" modal.
func DrawGeneratingOverlay(s tcell.Screen, sw, sh, n int, fill tcell.Style) {
	if n <= 0 {
		return
	}
	lines := []string{fmt.Sprintf(" Creating Level %d×%d... ", n, n)}
	drawBoxOverlay(s, sw, sh, lines, fill.Foreground(tcell.ColorWhite).Background(tcell.ColorNavy))
}

// DrawHelpOverlay paints the centered help modal.
func DrawHelpOverlay(s tcell.Screen, sw, sh int, fill tcell.Style) {
	lines := []string{
		" Arrows — TUI puzzle",
		"",
		" Fire an arrow (space/enter) to slide it off the board",
		" along its direction if the path is empty. If another",
		" arrow blocks the path, you lose a life.",
		"",
		" Win by clearing all arrows. Lose if lives hit 0.",
		"",
		" Any key closes this help.",
	}
	drawBoxOverlay(s, sw, sh, lines, fill.Foreground(tcell.ColorWhite).Background(tcell.ColorNavy))
}

// DrawModalOverlay paints the centered outcome modal.
func DrawModalOverlay(s tcell.Screen, sw, sh int, o *ModalOverlay, fill tcell.Style) {
	if o == nil || len(o.Lines) == 0 {
		return
	}
	bg := tcell.ColorDarkOliveGreen
	if !o.Positive {
		bg = tcell.ColorDarkRed
	}
	drawBoxOverlay(s, sw, sh, o.Lines, fill.Foreground(tcell.ColorWhite).Background(bg))
}

func drawBoxOverlay(s tcell.Screen, sw, sh int, lines []string, st tcell.Style) {
	boxW := 0
	for _, ln := range lines {
		if w := len([]rune(ln)); w > boxW {
			boxW = w
		}
	}
	boxW += 4
	boxH := len(lines) + 4
	ox := (sw - boxW) / 2
	oy := (sh - boxH) / 2
	if ox < 0 {
		ox = 0
	}
	if oy < 0 {
		oy = 0
	}

	for j := 0; j < boxH; j++ {
		for i := 0; i < boxW && i < sw; i++ {
			x, y := ox+i, oy+j
			if x < 0 || y < 0 || y >= sh {
				continue
			}
			var r rune = ' '
			if j == 0 && i == 0 {
				r = '┌'
			} else if j == 0 && i == boxW-1 {
				r = '┐'
			} else if j == boxH-1 && i == 0 {
				r = '└'
			} else if j == boxH-1 && i == boxW-1 {
				r = '┘'
			} else if j == 0 || j == boxH-1 {
				r = '─'
			} else if i == 0 || i == boxW-1 {
				r = '│'
			}
			s.SetContent(x, y, r, nil, st)
		}
	}
	for li, ln := range lines {
		DrawStr(s, ox+2, oy+2+li, ox+boxW-1, ln, st)
	}
}

// DrawStr draws a clipped string on one terminal row.
func DrawStr(s tcell.Screen, x0, y, maxW int, text string, st tcell.Style) {
	col := x0
	for _, r := range text {
		if col >= maxW {
			break
		}
		s.SetContent(col, y, r, nil, st)
		col++
	}
}
