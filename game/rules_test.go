package game

import "testing"

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

func TestRayEscapes_blocked(t *testing.T) {
	// Left snake vertical; right column head at bottom fires north into │.
	b, err := ParseLevelString("▲│\n│▲")
	if err != nil {
		t.Fatal(err)
	}
	if !RayEscapes(b, 0, 0) {
		t.Fatal("left head should still escape north")
	}
	if RayEscapes(b, 1, 1) {
		t.Fatal("right head ray north hits wire at (1,0)")
	}
}

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

func TestTryFire_blockedLosesLife(t *testing.T) {
	b, err := ParseLevelString("▲│\n│▲")
	if err != nil {
		t.Fatal(err)
	}
	g := NewGame(b, 2, "t")
	if TryFire(g, 1, 1) != FireBlocked {
		t.Fatal("expected blocked")
	}
	if g.Lives != 1 || g.Board.NonEmptyCount() != 4 {
		t.Fatalf("lives=%d cells=%d", g.Lives, g.Board.NonEmptyCount())
	}
}

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

func TestLost(t *testing.T) {
	b, err := ParseLevelString("▲│\n│▲")
	if err != nil {
		t.Fatal(err)
	}
	g := NewGame(b, 1, "t")
	_ = TryFire(g, 1, 1)
	if !g.Lost() {
		t.Fatal("expected lost")
	}
}

func TestParseLevel_asciiHeads(t *testing.T) {
	b, err := ParseLevelString("^\n│")
	if err != nil {
		t.Fatal(err)
	}
	if b.At(0, 0).R != '▲' {
		t.Fatal(b.At(0, 0).R)
	}
}
