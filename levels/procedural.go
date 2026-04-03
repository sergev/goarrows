package levels

import (
	"fmt"
	"math/rand/v2"

	"goarrows/game"
)

// ProceduralLevelCount is the nominal pack length for UI modulo and titles.
const ProceduralLevelCount = 1 << 20

// proceduralGenMaxTries is how many distinct base seeds (seed, seed+1, …) we try per level
// when GenerateBoard fails, before giving up.
const proceduralGenMaxTries = 512

type procMemo struct {
	b    game.Board
	name string
	err  error
}

type proceduralSource struct {
	seed int64
	memo map[int]procMemo
}

// newProceduralSource creates a memoizing generator for on-demand procedural boards.
func newProceduralSource(seed int64) *proceduralSource {
	return &proceduralSource{
		seed: seed,
		memo: make(map[int]procMemo),
	}
}

// levelAt builds or returns cached level i: (i+3)×(i+3) board, trying successive RNG seeds on failure.
func (p *proceduralSource) levelAt(i int) (game.Board, string, error) {
	if i < 0 {
		return game.Board{}, "", fmt.Errorf("negative level index")
	}
	if m, hit := p.memo[i]; hit {
		if m.err != nil {
			return game.Board{}, m.name, m.err
		}
		return m.b, m.name, nil
	}
	n := i + 3
	name := fmt.Sprintf("Level %d (%d×%d)", i+1, n, n)
	var b game.Board
	var err error
	for delta := int64(0); delta < proceduralGenMaxTries; delta++ {
		rng := levelRNG(p.seed+delta, i)
		b, err = game.GenerateBoard(n, n, rng)
		if err == nil {
			p.memo[i] = procMemo{b: b, name: name}
			return b, name, nil
		}
	}
	p.memo[i] = procMemo{name: name, err: err}
	return game.Board{}, name, err
}

// levelRNG is deterministic for a given (seed, level index).
func levelRNG(seed int64, idx int) *rand.Rand {
	s0 := uint64(seed) ^ uint64(uint32(idx))*0x9E3779B1
	s1 := uint64(idx)*0xC6A4A7935BD1E995 + uint64(seed)
	if s1%2 == 0 {
		s1++
	}
	return rand.New(rand.NewPCG(s0, s1))
}
