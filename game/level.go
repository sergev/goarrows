package game

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ParseLevel parses equal-length lines into a board and validates (full coverage, paths).
// Allowed: '.' ' ' empty (rejected by ValidateBoard), wires ─│┌┐└┘, heads ^v<> / ▲▼◀▶.
func ParseLevel(lines []string) (Board, error) {
	if len(lines) == 0 {
		return Board{}, fmt.Errorf("level: no rows")
	}
	w := utf8.RuneCountInString(lines[0])
	for i, line := range lines {
		if utf8.RuneCountInString(line) != w {
			return Board{}, fmt.Errorf("level: row %d length %d, want %d", i, utf8.RuneCountInString(line), w)
		}
	}
	h := len(lines)
	b := NewBoard(w, h)
	for y, line := range lines {
		x := 0
		for _, r := range line {
			c, err := parseCellRune(r)
			if err != nil {
				return Board{}, fmt.Errorf("level: row %d col %d: %w", y, x, err)
			}
			b.Set(x, y, c)
			x++
		}
	}
	if err := ValidateBoard(b); err != nil {
		return Board{}, err
	}
	return b, nil
}

// parseCellRune maps one ASCII/Unicode puzzle character to a Cell (empty, wire, or normalized head).
func parseCellRune(r rune) (Cell, error) {
	switch r {
	case '.', ' ':
		return Cell{}, nil
	case '─', '│', '┌', '┐', '└', '┘':
		return Cell{R: r}, nil
	case '^', '▲', 'v', 'V', '▼', '<', '◀', '>', '▶':
		return Cell{R: normalizeHeadRune(r)}, nil
	default:
		return Cell{}, fmt.Errorf("invalid rune %q", r)
	}
}

// normalizeHeadRune maps ASCII arrow heads to the Unicode heads used internally.
func normalizeHeadRune(r rune) rune {
	switch r {
	case '^':
		return '▲'
	case 'v', 'V':
		return '▼'
	case '<':
		return '◀'
	case '>':
		return '▶'
	default:
		return r
	}
}

// ParseLevelString splits on newlines and drops trailing empty lines.
func ParseLevelString(s string) (Board, error) {
	s = strings.TrimRight(s, "\n")
	lines := strings.Split(s, "\n")
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return ParseLevel(lines)
}
