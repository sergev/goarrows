package game

import (
	"strings"
	"testing"
)

// TestValidateBoard_mismatchedNeighbor rejects a head wired to an incompatible neighbor.
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

// TestValidateBoard_emptyRejected rejects full-validation boards that contain empty cells.
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

// TestValidateBoard_twoHeadsOneComponent rejects two heads in one connected component.
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

// TestValidatePartialBoard_okSparseArrow accepts a small arrow with background empties.
func TestValidatePartialBoard_okSparseArrow(t *testing.T) {
	// ▲ at (0,0), │ below — rest empty
	b := NewBoard(2, 2)
	b.Set(0, 0, Cell{R: '▲'})
	b.Set(0, 1, Cell{R: '│'})
	if err := ValidatePartialBoard(b); err != nil {
		t.Fatal(err)
	}
}

// TestValidatePartialBoard_noCells rejects a board with no arrow cells.
func TestValidatePartialBoard_noCells(t *testing.T) {
	b := NewBoard(2, 2)
	if err := ValidatePartialBoard(b); err == nil {
		t.Fatal("expected error for empty board")
	}
}

// TestValidatePartialBoard_twoHeadsOneComponent rejects two heads sharing one component on partial boards.
func TestValidatePartialBoard_twoHeadsOneComponent(t *testing.T) {
	b := NewBoard(2, 1)
	b.Set(0, 0, Cell{R: '▶'})
	b.Set(1, 0, Cell{R: '▲'})
	err := ValidatePartialBoard(b)
	if err == nil {
		t.Fatal("expected component error")
	}
}
