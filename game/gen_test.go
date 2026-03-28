package game

import (
	"math/rand/v2"
	"testing"
)

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

func TestGenerateFullBoardValidateAndSolvable(t *testing.T) {
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
			if err := ValidateBoard(b); err != nil {
				t.Fatalf("n=%d seed=%d validate: %v", n, seed, err)
			}
		}
	}
}

func TestGenerateFullBoardLargeSmoke(t *testing.T) {
	// Tuned for `go test -timeout 10s ./...` (see Makefile).
	sizes := []int{10}
	for _, n := range sizes {
		rng := rand.New(rand.NewPCG(42, uint64(n)*99991+17))
		b, err := GenerateFullBoard(n, n, rng)
		if err != nil {
			t.Fatalf("n=%d: %v", n, err)
		}
		if b.NonEmptyCount() != n*n {
			t.Fatalf("n=%d: want full coverage", n)
		}
		if err := ValidateBoard(b); err != nil {
			t.Fatalf("n=%d validate: %v", n, err)
		}
	}
}

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

func TestGenerateFullBoardPlayfulnessSmoke(t *testing.T) {
	rng := rand.New(rand.NewPCG(2024, 303))
	b, err := GenerateFullBoard(10, 10, rng)
	if err != nil {
		t.Fatal(err)
	}
	esc := countInitialRayEscapes(b)
	if esc < 0 || esc > 80 {
		t.Fatalf("implausible initial escape count: %d", esc)
	}
}

func TestGenerateFullBoardVariedHeadCount(t *testing.T) {
	// Generator should not collapse to exactly two long snakes on medium boards; expect 3+ heads often.
	const n = 10
	ge3 := 0
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
		if countHeads(b) >= 3 {
			ge3++
		}
	}
	minOK := 2
	if seeds >= 8 {
		minOK = 3
	}
	if ge3 < minOK {
		t.Fatalf("want at least %d/%d boards with 3+ arrow heads, got %d", minOK, seeds, ge3)
	}
}

func TestGenerateFullBoardMultipleComponents(t *testing.T) {
	// Greedy / K-band paths should yield more than one arrowhead.
	cases := []struct {
		w, h int
	}{
		{8, 8},
		{10, 10},
		{6, 9},
	}
	for _, tc := range cases {
		rng := rand.New(rand.NewPCG(uint64(tc.w*97+tc.h), uint64(tc.w*tc.h)+13))
		b, err := GenerateFullBoard(tc.w, tc.h, rng)
		if err != nil {
			t.Fatalf("%d×%d: %v", tc.w, tc.h, err)
		}
		h := countHeads(b)
		if h < 2 {
			t.Fatalf("%d×%d: want at least 2 arrow heads, got %d", tc.w, tc.h, h)
		}
	}
}

// VerifySolvable is exponential in board size; spot-check only tiny boards.
func TestGenerateFullBoardVerifySolvableTiny(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 11))
	b, err := GenerateFullBoard(3, 3, rng)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifySolvable(b) {
		t.Fatal("expected VerifySolvable on 3×3")
	}
}
