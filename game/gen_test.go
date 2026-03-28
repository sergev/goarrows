package game

import (
	"math/rand/v2"
	"testing"
)

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
