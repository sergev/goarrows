package game

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ParseLevel parses equal-length lines into a board. Runes:
// '.' space ' ' = empty; '^' 'v' '<' '>' or '▲' '▼' '◀' '▶' = arrows.
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
	return b, nil
}

func parseCellRune(r rune) (Cell, error) {
	switch r {
	case '.', ' ':
		return Cell{Empty: true}, nil
	case '^', '▲':
		return Cell{Dir: North}, nil
	case 'v', 'V', '▼':
		return Cell{Dir: South}, nil
	case '<', '◀':
		return Cell{Dir: West}, nil
	case '>', '▶':
		return Cell{Dir: East}, nil
	default:
		return Cell{}, fmt.Errorf("invalid rune %q", r)
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
