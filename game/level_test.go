package game

import (
	"strings"
	"testing"
)

func TestValidateBoard_mismatchedNeighbor(t *testing.T) {
	// ▲ with no body link (│ below has no north to ▲ if we use wrong glyph)
	b := NewBoard(1, 2)
	b.Set(0, 0, Cell{R: '▲'})
	b.Set(0, 1, Cell{R: '─'}) // horizontal wire cannot connect north
	err := ValidateBoard(b)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateBoard_emptyRejected(t *testing.T) {
	b := NewBoard(1, 1)
	b.Set(0, 0, Cell{R: '▲'})
	// not full coverage if we could parse - use manual board with empty
	b2 := NewBoard(2, 1)
	b2.Set(0, 0, Cell{R: '▲'})
	b2.Set(1, 0, Cell{})
	err := ValidateBoard(b2)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateBoard_twoHeadsOneComponent(t *testing.T) {
	// two heads adjacent without proper separation - invalid graph
	b := NewBoard(2, 1)
	b.Set(0, 0, Cell{R: '▶'})
	b.Set(1, 0, Cell{R: '▲'})
	err := ValidateBoard(b)
	if err == nil {
		t.Fatal("expected component head count error")
	}
}
