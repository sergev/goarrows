package game

import (
	"testing"
	"unicode/utf8"
)

// boardFromLinesNoValidate builds a board like ParseLevel but skips ValidateBoard.
// Used for cases where ray/fire behavior is tested on layouts that are not valid
// full puzzles under strict graph rules (e.g. two components with a head firing across a foreign wire).
func boardFromLinesNoValidate(lines []string) Board {
	w := utf8.RuneCountInString(lines[0])
	b := NewBoard(w, len(lines))
	for y, line := range lines {
		x := 0
		for _, r := range line {
			c, err := parseCellRune(r)
			if err != nil {
				panic(err)
			}
			b.Set(x, y, c)
			x++
		}
	}
	return b
}

// TestRayEscapes_headOnly checks RayEscapes on a head vs wire on a minimal valid level.
func TestRayEscapes_headOnly(t *testing.T) {
	b, err := ParseLevelString("▲\n│")
	if err != nil {
		t.Fatal(err)
	}
	if RayEscapes(b, 0, 1) {
		t.Fatal("wire must not ray-escape")
	}
	if !RayEscapes(b, 0, 0) {
		t.Fatal("head should escape north")
	}
}

// TestRayEscapes_blocked uses a non-valid layout to assert blocked vs open rays.
func TestRayEscapes_blocked(t *testing.T) {
	// Left snake vertical; right column head at bottom fires north into │.
	b := boardFromLinesNoValidate([]string{"▲│", "│▲"})
	if !RayEscapes(b, 0, 0) {
		t.Fatal("left head should still escape north")
	}
	if RayEscapes(b, 1, 1) {
		t.Fatal("right head ray north hits wire at (1,0)")
	}
}

// TestTryFire_wireNoOp ensures firing a non-head cell is a no-op.
func TestTryFire_wireNoOp(t *testing.T) {
	b, err := ParseLevelString("▲\n│")
	if err != nil {
		t.Fatal(err)
	}
	g := NewGame(b, 2, "t")
	if TryFire(g, 0, 1) != FireNone {
		t.Fatal("firing wire")
	}
	if g.Board.NonEmptyCount() != 2 {
		t.Fatal("board unchanged")
	}
}

// TestTryFire_clearsFullPath clears a vertical arrow and wins.
func TestTryFire_clearsFullPath(t *testing.T) {
	b, err := ParseLevelString("▲\n│")
	if err != nil {
		t.Fatal(err)
	}
	g := NewGame(b, 1, "t")
	if TryFire(g, 0, 0) != FireCleared {
		t.Fatal("expected cleared")
	}
	if g.Board.NonEmptyCount() != 0 || !g.Won() {
		t.Fatal("path should be fully cleared")
	}
}

// TestTryFire_blockedLosesLife checks FireBlocked decrements lives and leaves the board intact.
func TestTryFire_blockedLosesLife(t *testing.T) {
	b := boardFromLinesNoValidate([]string{"▲│", "│▲"})
	g := NewGame(b, 2, "t")
	if TryFire(g, 1, 1) != FireBlocked {
		t.Fatal("expected blocked")
	}
	if g.Lives != 1 || g.Board.NonEmptyCount() != 4 {
		t.Fatalf("lives=%d cells=%d", g.Lives, g.Board.NonEmptyCount())
	}
}

// TestTryFire_horizontalClearsAll clears a single horizontal polyline in one shot.
func TestTryFire_horizontalClearsAll(t *testing.T) {
	b, err := ParseLevelString("──▶")
	if err != nil {
		t.Fatal(err)
	}
	g := NewGame(b, 1, "t")
	if TryFire(g, 2, 0) != FireCleared {
		t.Fatal("head at east end")
	}
	if !g.Won() {
		t.Fatal("won")
	}
}

// TestLost checks Lost after a blocked shot consumes the last life.
func TestLost(t *testing.T) {
	b := boardFromLinesNoValidate([]string{"▲│", "│▲"})
	g := NewGame(b, 1, "t")
	_ = TryFire(g, 1, 1)
	if !g.Lost() {
		t.Fatal("expected lost")
	}
}

// TestParseLevel_asciiHeads verifies ASCII head runes normalize to Unicode in ParseLevelString.
func TestParseLevel_asciiHeads(t *testing.T) {
	b, err := ParseLevelString("^\n│")
	if err != nil {
		t.Fatal(err)
	}
	if b.At(0, 0).R != '▲' {
		t.Fatal(b.At(0, 0).R)
	}
}
