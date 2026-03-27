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
		seeds := uint64(20)
		if n >= 6 {
			seeds = 5
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
			// Backtracking solvability check is exponential; keep to modest sizes in CI.
			if n <= 5 && !VerifySolvable(b) {
				t.Fatalf("n=%d seed=%d: not solvable", n, seed)
			}
		}
	}
}
