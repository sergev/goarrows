package levels

import (
	"fmt"

	"goarrows/game"
)
// Pack is an ordered set of levels from procedural generation
// (when proc is non-nil) or from test-built board/name slices.
type Pack struct {
	Names   []string
	Boards  []game.Board
	proc    *proceduralSource
}

// NewProceduralPack returns a pack with unbounded levels: size (i+3)×(i+3)
// for index i, deterministic per seed.
func NewProceduralPack(seed int64) *Pack {
	return &Pack{proc: newProceduralSource(seed)}
}

// Len returns the number of levels (large constant for procedural packs).
func (p *Pack) Len() int {
	if p.proc != nil {
		return ProceduralLevelCount
	}
	return len(p.Boards)
}

// ProceduralSideLen returns the N in N×N for procedural level index i, or 0 if the pack is not procedural.
func (p *Pack) ProceduralSideLen(i int) int {
	if p.proc == nil {
		return 0
	}
	return i + 3
}

// ProceduralLevelReady reports whether level i is already generated and cached (always true for file-based packs).
func (p *Pack) ProceduralLevelReady(i int) bool {
	if p.proc == nil {
		return true
	}
	_, ok := p.proc.memo[i]
	return ok
}

// LevelAt returns the board and display name for index i.
func (p *Pack) LevelAt(i int) (game.Board, string, error) {
	if p.proc != nil {
		return p.proc.levelAt(i)
	}
	if i < 0 || i >= len(p.Boards) {
		return game.Board{}, "", fmt.Errorf("level index %d out of range [0,%d)", i, len(p.Boards))
	}
	return p.Boards[i], p.Names[i], nil
}
