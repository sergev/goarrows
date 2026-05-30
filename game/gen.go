package game

import (
	"math/rand/v2"
)

// GenGrow names the procedural algorithm. The label is retained for backward
// compatibility with stored level identifiers; the implementation is the
// constructive reverse-order generator in gen_reverse.go.
const GenGrow = "grow"

// GenerateBoard fills a w×h grid by placing arrows in reverse firing order so
// every board it returns is solvable by construction.
func GenerateBoard(w, h int, rng *rand.Rand) (Board, error) {
	return generateFullBoardReverse(w, h, rng)
}

// targetArrowCountForSide returns how many arrow polylines to use for an N×N layer:
// N < 6 → N; N < 10 → N*N/6; otherwise → N*N/10 (integer division).
func targetArrowCountForSide(n int) int {
	switch {
	case n < 6:
		return n
	case n < 10:
		return n * n / 6
	default:
		return n * n / 10
	}
}

// clampArrowCount caps the target polyline count to [1, wh/2] so seeds fit on the grid.
func clampArrowCount(targetArrows, wh int) int {
	maxArrows := wh / 2
	if maxArrows < 1 {
		maxArrows = 1
	}
	if targetArrows > maxArrows {
		targetArrows = maxArrows
	}
	if targetArrows < 1 {
		targetArrows = 1
	}
	return targetArrows
}

// genScratch holds buffers reused by the constructive reverse-order generator
// across restarts. All slices are sized for the worst case (nHeads heads on a
// w×h grid); buffers are reset via clear() or slice-resize at the top of each
// restart in generateFullBoardReverse.
type genScratch struct {
	glyphAt   []rune     // size w*h — running partial board (rune at each cell, 0 == empty)
	headXs    []int      // grows to nHeads — head x-coordinates in placement order
	headYs    []int      // grows to nHeads — head y-coordinates
	headByDir [4][]Point // heads grouped by fire direction (for same-dir spread lookup)
	headsBuf  []Point    // reused Point list passed to growPlayfulEnoughHeads
	cellCands []cellCand // reused enumerateAllCandidates output
	stepBuf   []Point    // reused growBodyReverse 4-neighbor candidate list
}

// newGenScratch allocates the per-call scratch buffers used by the constructive
// reverse-order generator.
func newGenScratch(w, h, nHeads int) *genScratch {
	wh := w * h
	sc := &genScratch{
		glyphAt:   make([]rune, wh),
		headXs:    make([]int, 0, nHeads),
		headYs:    make([]int, 0, nHeads),
		headsBuf:  make([]Point, 0, nHeads),
		cellCands: make([]cellCand, 0, 4*wh),
		stepBuf:   make([]Point, 0, 4),
	}
	for i := range sc.headByDir {
		sc.headByDir[i] = make([]Point, 0, nHeads)
	}
	return sc
}

// growPlayfulEnough rejects boards that are too easy at the start: at most half the heads
// may have a clear firing ray (RayEscapes). For a single head the check is skipped so
// solvable tiny cases are still possible.
func growPlayfulEnough(b Board) bool {
	var heads []Point
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			if b.At(x, y).IsHead() {
				heads = append(heads, Point{x, y})
			}
		}
	}
	return growPlayfulEnoughHeads(b, heads)
}

// growPlayfulEnoughHeads is the same predicate as growPlayfulEnough but skips the
// full-board scan when the caller already knows the head positions.
func growPlayfulEnoughHeads(b Board, heads []Point) bool {
	total := len(heads)
	if total <= 1 {
		return true
	}
	fireable := 0
	for _, h := range heads {
		if RayEscapes(b, h.X, h.Y) {
			fireable++
		}
	}
	return 2*fireable <= total
}
