package game

import (
	"fmt"
	"math/rand/v2"
)

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

// generateFullBoardGrow seeds arrow heads with a one-cell body (count from targetArrowCountForSide),
// extends tails at random until stuck, then accepts only if ValidatePartialBoard,
// growPlayfulEnough (at most half the heads have a clear shot at start), and
// VerifyGreedyFirstClearsBoard succeed.
func generateFullBoardGrow(w, h int, rng *rand.Rand) (Board, error) {
	if w <= 0 || h <= 0 {
		return Board{}, fmt.Errorf("gen: invalid size %d×%d", w, h)
	}
	wh := w * h
	if wh < 2 {
		return Board{}, fmt.Errorf("gen: need at least 2 cells (got %d×%d)", w, h)
	}

	n := minInt(w, h)
	nHeads := clampArrowCount(targetArrowCountForSide(n), wh)
	maxTries := 8000 + 100*wh
	if maxTries > 60000 {
		maxTries = 60000
	}
	for attempt := 0; attempt < maxTries; attempt++ {
		r := rand.New(rand.NewPCG(rng.Uint64(), rng.Uint64()))
		paths, ok := tryGrowPartition(w, h, nHeads, r)
		if !ok {
			continue
		}
		b, err := boardFromPaths(paths, w, h)
		if err != nil {
			continue
		}
		if err := ValidatePartialBoard(b); err != nil {
			continue
		}
		if !growPlayfulEnough(b) {
			continue
		}
		if !VerifyGreedyFirstClearsBoard(b) {
			continue
		}
		return b, nil
	}
	return Board{}, fmt.Errorf("gen: could not build grow board for %d×%d", w, h)
}

// growPlayfulEnough rejects boards that are too easy at the start: at most half the heads
// may have a clear firing ray (RayEscapes). For a single head the check is skipped so
// solvable tiny cases are still possible.
func growPlayfulEnough(b Board) bool {
	total := 0
	fireable := 0
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			if !b.At(x, y).IsHead() {
				continue
			}
			total++
			if RayEscapes(b, x, y) {
				fireable++
			}
		}
	}
	if total <= 1 {
		return true
	}
	return 2*fireable <= total
}

// tryGrowPartition builds initial seed paths then repeatedly extends tails until no move remains.
func tryGrowPartition(w, h, nHeads int, rng *rand.Rand) ([][]point, bool) {
	paths, ok := seedGrowPaths(w, h, nHeads, rng)
	if !ok {
		return nil, false
	}

	occupied := make([]bool, w*h)
	for _, path := range paths {
		for _, p := range path {
			occupied[p.y*w+p.x] = true
		}
	}

	for {
		extended := false
		order := rng.Perm(len(paths))
		for _, pi := range order {
			path := paths[pi]
			if len(path) < 2 {
				return nil, false
			}
			tail := path[len(path)-1]
			prev := path[len(path)-2]
			pathSet := make(map[point]struct{}, len(path)+1)
			for _, p := range path {
				pathSet[p] = struct{}{}
			}
			cands := neighborPoints(tail, prev, w, h, occupied, pathSet)
			hx, hy := path[0].x, path[0].y
			fire := oppositeDirGen(dirFromTo(hx, hy, path[1].x, path[1].y))
			write := 0
			for _, c := range cands {
				if !cellOnOpenRayFromHead(hx, hy, fire, c.x, c.y, w, h) {
					cands[write] = c
					write++
				}
			}
			cands = cands[:write]
			if len(cands) == 0 {
				continue
			}
			next := pickBiasedTailStep(prev, tail, cands, rng, growStraightChance10)
			path = append(path, next)
			paths[pi] = path
			occupied[next.y*w+next.x] = true
			extended = true
		}
		if !extended {
			break
		}
	}

	return paths, true
}

// seedGrowPaths places nHeads disjoint two-cell paths (head + one body) with random orientation.
func seedGrowPaths(w, h, nHeads int, rng *rand.Rand) ([][]point, bool) {
	wh := w * h
	occupied := make([]bool, wh)
	var heads []point
	var paths [][]point

	maxSeedAttempts := 400 * nHeads
	if maxSeedAttempts < 800 {
		maxSeedAttempts = 800
	}

	for k := 0; k < nHeads; k++ {
		placed := false
		for attempt := 0; attempt < maxSeedAttempts; attempt++ {
			hx := rng.IntN(w)
			hy := rng.IntN(h)
			if occupied[hy*w+hx] {
				continue
			}
			fire := Direction(rng.IntN(4))
			if rayHitsPreviousHead(hx, hy, fire, heads, w, h) {
				continue
			}
			bdx, bdy := Delta(oppositeDir(fire))
			bx, by := hx+bdx, hy+bdy
			if bx < 0 || bx >= w || by < 0 || by >= h {
				continue
			}
			if occupied[by*w+bx] {
				continue
			}
			path := []point{{hx, hy}, {bx, by}}
			occupied[hy*w+hx] = true
			occupied[by*w+bx] = true
			heads = append(heads, point{hx, hy})
			paths = append(paths, path)
			placed = true
			break
		}
		if !placed {
			return nil, false
		}
	}
	return paths, true
}

// rayHitsPreviousHead reports whether the open ray from head (hx,hy) in fire direction passes any prior head.
func rayHitsPreviousHead(hx, hy int, fire Direction, heads []point, w, h int) bool {
	dx, dy := Delta(fire)
	for cx, cy := hx+dx, hy+dy; cx >= 0 && cx < w && cy >= 0 && cy < h; cx, cy = cx+dx, cy+dy {
		for _, hp := range heads {
			if hp.x == cx && hp.y == cy {
				return true
			}
		}
	}
	return false
}

// boardFromPaths rasterizes polylines into a Board via paintPath (each path must be disjoint).
func boardFromPaths(paths [][]point, w, h int) (Board, error) {
	grid := make([]rune, w*h)
	for _, path := range paths {
		if err := paintPath(grid, w, path); err != nil {
			return Board{}, err
		}
	}
	b := NewBoard(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			b.Set(x, y, Cell{R: grid[y*w+x]})
		}
	}
	return b, nil
}
