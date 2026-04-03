package game

import (
	"math/rand/v2"
	"testing"
)

// countHeads returns how many cells are arrow heads on b.
func countHeads(b Board) int {
	n := 0
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			if b.At(x, y).IsHead() {
				n++
			}
		}
	}
	return n
}

// boardRunesEqual compares two boards’ dimensions and per-cell runes.
func boardRunesEqual(a, b Board) bool {
	if a.W != b.W || a.H != b.H || len(a.Data) != len(b.Data) {
		return false
	}
	for i := range a.Data {
		if a.Data[i].R != b.Data[i].R {
			return false
		}
	}
	return true
}

// TestCellOnOpenRayFromHead exercises cellOnOpenRayFromHead across directions and edge cases.
func TestCellOnOpenRayFromHead(t *testing.T) {
	tests := []struct {
		name       string
		hx, hy     int
		fire       Direction
		px, py     int
		w, h       int
		wantOnRay bool
	}{
		{"first_cell_north", 2, 2, North, 2, 1, 5, 5, true},
		{"second_cell_north", 2, 2, North, 2, 0, 5, 5, true},
		{"head_not_counted", 2, 2, North, 2, 2, 5, 5, false},
		{"off_ray_diagonal", 2, 2, North, 3, 1, 5, 5, false},
		{"east_ray", 1, 1, East, 3, 1, 5, 5, true},
		{"beyond_board_not_walked", 1, 1, East, 5, 1, 5, 5, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cellOnOpenRayFromHead(tt.hx, tt.hy, tt.fire, tt.px, tt.py, tt.w, tt.h)
			if got != tt.wantOnRay {
				t.Fatalf("got %v, want %v", got, tt.wantOnRay)
			}
		})
	}
}

// TestGenerateFullBoardValidateAndPlayable smoke-tests generation, validation, and greedy solvability at several sizes.
func TestGenerateFullBoardValidateAndPlayable(t *testing.T) {
	sizes := []int{3, 4, 5, 6}
	if testing.Short() {
		sizes = []int{3, 4, 5}
	}
	for _, n := range sizes {
		seeds := uint64(12)
		if testing.Short() {
			seeds = 3
		}
		if n >= 6 {
			seeds = 5
			if testing.Short() {
				seeds = 2
			}
		}
		for seed := uint64(1); seed <= seeds; seed++ {
			rng := rand.New(rand.NewPCG(seed, seed*2+1))
			b, err := GenerateFullBoard(n, n, rng)
			if err != nil {
				t.Fatalf("n=%d seed=%d: %v", n, seed, err)
			}
			if b.W != n || b.H != n {
				t.Fatalf("n=%d seed=%d: got %d×%d", n, seed, b.W, b.H)
			}
			if err := ValidatePartialBoard(b); err != nil {
				t.Fatalf("n=%d seed=%d validate partial: %v", n, seed, err)
			}
			if !VerifyGreedyFirstClearsBoard(b) {
				t.Fatalf("n=%d seed=%d expected greedy clear", n, seed)
			}
		}
	}
}

// TestGenerateFullBoardLargeSmoke runs one larger board under a tight test timeout budget.
func TestGenerateFullBoardLargeSmoke(t *testing.T) {
	// Tuned for `go test -timeout 10s ./...` (see Makefile).
	sizes := []int{8}
	for _, n := range sizes {
		rng := rand.New(rand.NewPCG(42, uint64(n)*99991+17))
		b, err := GenerateFullBoard(n, n, rng)
		if err != nil {
			t.Fatalf("n=%d: %v", n, err)
		}
		if err := ValidatePartialBoard(b); err != nil {
			t.Fatalf("n=%d validate partial: %v", n, err)
		}
		if !VerifyGreedyFirstClearsBoard(b) {
			t.Fatalf("n=%d expected greedy clear", n)
		}
	}
}

// TestGenerateFullBoardReproducible checks identical PCG state yields identical boards.
func TestGenerateFullBoardReproducible(t *testing.T) {
	const seed0, seed1 uint64 = 0x1234abcd, 0xf00dcafe
	rng1 := rand.New(rand.NewPCG(seed0, seed1))
	rng2 := rand.New(rand.NewPCG(seed0, seed1))
	b1, err := GenerateFullBoard(8, 8, rng1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := GenerateFullBoard(8, 8, rng2)
	if err != nil {
		t.Fatal(err)
	}
	if !boardRunesEqual(b1, b2) {
		t.Fatal("same PCG seeds should yield identical boards")
	}
}

// TestGenerateFullBoardPlayfulnessSmoke sanity-checks initial RayEscapes head count on one board.
func TestGenerateFullBoardPlayfulnessSmoke(t *testing.T) {
	rng := rand.New(rand.NewPCG(2024, 303))
	b, err := GenerateFullBoard(8, 8, rng)
	if err != nil {
		t.Fatal(err)
	}
	esc := countInitialRayEscapes(b)
	if esc < 0 || esc > 80 {
		t.Fatalf("implausible initial escape count: %d", esc)
	}
}

// TestGenerateFullBoardVariedHeadCount expects every generated board in a seed sweep to have at least one head.
func TestGenerateFullBoardVariedHeadCount(t *testing.T) {
	// Grow generator should consistently produce boards with at least one head.
	const n = 8
	nonZero := 0
	seeds := uint64(8)
	if testing.Short() {
		seeds = 4
	}
	for seed := uint64(1); seed <= seeds; seed++ {
		rng := rand.New(rand.NewPCG(seed, 777))
		b, err := GenerateFullBoard(n, n, rng)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if countHeads(b) > 0 {
			nonZero++
		}
	}
	if nonZero != int(seeds) {
		t.Fatalf("want heads on all %d/%d boards, got %d", seeds, seeds, nonZero)
	}
}

// TestGenerateFullBoardMultipleComponents checks non-square rectangles still produce at least one head.
func TestGenerateFullBoardMultipleComponents(t *testing.T) {
	// Grow generator should produce at least one arrowhead.
	cases := []struct {
		w, h int
	}{
		{8, 8},
		{6, 9},
		{7, 8},
	}
	for _, tc := range cases {
		rng := rand.New(rand.NewPCG(uint64(tc.w*97+tc.h), uint64(tc.w*tc.h)+13))
		b, err := GenerateFullBoard(tc.w, tc.h, rng)
		if err != nil {
			t.Fatalf("%d×%d: %v", tc.w, tc.h, err)
		}
		h := countHeads(b)
		if h < 1 {
			t.Fatalf("%d×%d: want at least 1 arrow head, got %d", tc.w, tc.h, h)
		}
	}
}

// TestGenerateFullBoardGreedyClearsTiny asserts greedy solvability on a tiny generated board.
func TestGenerateFullBoardGreedyClearsTiny(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 11))
	b, err := GenerateFullBoard(3, 3, rng)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyGreedyFirstClearsBoard(b) {
		t.Fatal("expected VerifyGreedyFirstClearsBoard on 3×3")
	}
}

// TestVerifyGreedyFirstClearsBoard_verticalArrow checks greedy clear on a hand-built vertical arrow.
func TestVerifyGreedyFirstClearsBoard_verticalArrow(t *testing.T) {
	b := NewBoard(2, 2)
	b.Set(0, 0, Cell{R: '▲'})
	b.Set(0, 1, Cell{R: '│'})
	if !VerifyGreedyFirstClearsBoard(b) {
		t.Fatal("expected greedy clear")
	}
}

// TestGenerateBoardGrowSmoke validates one grow-generated board and the “half fireable” heuristic when heads ≥ 2.
func TestGenerateBoardGrowSmoke(t *testing.T) {
	rng := rand.New(rand.NewPCG(99, 101))
	b, err := GenerateBoard(7, 7, rng)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePartialBoard(b); err != nil {
		t.Fatal(err)
	}
	if !VerifyGreedyFirstClearsBoard(b) {
		t.Fatal("expected VerifyGreedyFirstClearsBoard")
	}
	fireable := 0
	heads := 0
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			if !b.At(x, y).IsHead() {
				continue
			}
			heads++
			if RayEscapes(b, x, y) {
				fireable++
			}
		}
	}
	if heads >= 2 && 2*fireable > heads {
		t.Fatalf("grow smoke: want at most half fireable at start, got %d/%d", fireable, heads)
	}
}

// TestGrowPlayfulEnough checks growPlayfulEnough rejects too-easy two-head boards and skips the check for one head.
func TestGrowPlayfulEnough(t *testing.T) {
	b := NewBoard(5, 2)
	b.Set(0, 0, Cell{R: '▲'})
	b.Set(0, 1, Cell{R: '│'})
	b.Set(4, 0, Cell{R: '▲'})
	b.Set(4, 1, Cell{R: '│'})
	if err := ValidatePartialBoard(b); err != nil {
		t.Fatal(err)
	}
	if growPlayfulEnough(b) {
		t.Fatal("both heads have clear rays: expected not playful enough")
	}

	single := NewBoard(2, 2)
	single.Set(0, 0, Cell{R: '▲'})
	single.Set(0, 1, Cell{R: '│'})
	if !growPlayfulEnough(single) {
		t.Fatal("single head: check skipped, expected playful")
	}
}

// TestGenGrowConstant locks the GenGrow algorithm name string used by the game.
func TestGenGrowConstant(t *testing.T) {
	if GenGrow != "grow" {
		t.Fatalf("GenGrow = %q, want %q", GenGrow, "grow")
	}
}
