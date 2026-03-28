package game

import (
	"math/rand/v2"
	"testing"
)

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
	sizes := []int{10, 20}
	if testing.Short() {
		sizes = []int{10}
	}
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
