package game

import "testing"

func TestRayEscapes_clearToNorthEdge(t *testing.T) {
	b, err := ParseLevelString("▲\n.\n.")
	if err != nil {
		t.Fatal(err)
	}
	if !RayEscapes(b, 0, 0) {
		t.Fatal("expected escape north")
	}
}

func TestRayEscapes_blockedByNeighbor(t *testing.T) {
	// ▲ at bottom; ray north hits ▶ before top edge.
	b, err := ParseLevelString(".\n▶\n▲")
	if err != nil {
		t.Fatal(err)
	}
	if RayEscapes(b, 0, 2) {
		t.Fatal("expected blocked")
	}
}

func TestRayEscapes_clearEast(t *testing.T) {
	b, err := ParseLevelString("..▶")
	if err != nil {
		t.Fatal(err)
	}
	if !RayEscapes(b, 2, 0) {
		t.Fatal("expected escape east")
	}
}

func TestTryFire_emptyNoOp(t *testing.T) {
	b, _ := ParseLevelString("..")
	g := NewGame(b, 3, "t")
	if TryFire(g, 0, 0) != FireNone {
		t.Fatal("empty should be FireNone")
	}
	if g.Lives != 3 {
		t.Fatalf("lives=%d", g.Lives)
	}
}

func TestTryFire_blockedLosesLife(t *testing.T) {
	b, _ := ParseLevelString(".\n▶\n▲")
	g := NewGame(b, 2, "t")
	if TryFire(g, 0, 2) != FireBlocked {
		t.Fatalf("got %v", TryFire(g, 0, 2))
	}
	if g.Lives != 1 {
		t.Fatalf("lives=%d", g.Lives)
	}
	if g.Board.ArrowCount() != 2 {
		t.Fatal("board should be unchanged")
	}
}

func TestTryFire_clearRemovesArrow(t *testing.T) {
	b, _ := ParseLevelString("▲\n.\n.")
	g := NewGame(b, 1, "t")
	if TryFire(g, 0, 0) != FireCleared {
		t.Fatal("expected cleared")
	}
	if g.Board.At(0, 0).Empty != true {
		t.Fatal("cell should be empty")
	}
	if !g.Won() {
		t.Fatal("should win after last arrow — need board with single arrow")
	}
}

func TestTryFire_winAfterLastArrow(t *testing.T) {
	b, _ := ParseLevelString("▲")
	g := NewGame(b, 1, "t")
	_ = TryFire(g, 0, 0)
	if !g.Won() {
		t.Fatal("won")
	}
}

func TestLost(t *testing.T) {
	b2, _ := ParseLevelString(".\n▶\n▲")
	g2 := NewGame(b2, 1, "t")
	_ = TryFire(g2, 0, 2)
	if !g2.Lost() {
		t.Fatal("expected lost")
	}
}

func TestParseLevel_unicodeColumns(t *testing.T) {
	b, err := ParseLevelString("▲▶")
	if err != nil {
		t.Fatal(err)
	}
	if b.W != 2 || b.At(0, 0).Dir != North || b.At(1, 0).Dir != East {
		t.Fatalf("%+v %+v", b.At(0, 0), b.At(1, 0))
	}
}

func TestParseLevel_ascii(t *testing.T) {
	b, err := ParseLevelString("^.>\n.v.")
	if err != nil {
		t.Fatal(err)
	}
	if b.W != 3 || b.H != 2 {
		t.Fatalf("%dx%d", b.W, b.H)
	}
	if b.At(0, 0).Empty || b.At(0, 0).Dir != North {
		t.Fatal(b.At(0, 0))
	}
}
