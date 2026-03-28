package game

import (
	"fmt"
	"math/rand/v2"
)

// placedComponent records one polyline placed during generation (head at path[0]).
// placementOrder is the order paths were committed: first entry is the first path
// placed on an empty board; forward play removes paths in reverse order.
type placedComponent struct {
	path []point
}

const (
	maxRestart      = 8000
	maxRestartLarge = 16000 // more attempts for bigger boards where greedy often needs retries
	maxGrowAttempts = 1200  // inner random trials in tryGrowPath
)

// GenerateFullBoard fills a w×h grid with disjoint arrow paths using reverse
// construction so that at every forward step at least one head can fire until
// the board is clear. It retries with the same RNG stream on failure.
func GenerateFullBoard(w, h int, rng *rand.Rand) (Board, error) {
	wh := w * h
	restarts := maxRestart
	if wh >= 30 {
		restarts = maxRestartLarge
	}
	for attempt := 0; attempt < restarts; attempt++ {
		occ := make([]bool, wh)
		grid := make([]rune, wh)
		emptyCount := wh
		var placementOrder []placedComponent
		okFill := true
		fillSteps := 0
		maxFillSteps := wh*wh + 80
		for emptyCount > 0 && okFill {
			fillSteps++
			if fillSteps > maxFillSteps {
				okFill = false
				break
			}
			if !tryPlaceOnePath(w, h, occ, grid, &emptyCount, rng, &placementOrder) {
				okFill = false
			}
		}

		if !okFill || emptyCount != 0 {
			continue
		}

		b := NewBoard(w, h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				b.Set(x, y, Cell{R: grid[y*w+x]})
			}
		}
		if err := ValidateBoard(b); err != nil {
			continue
		}
		if !verifyReverseConstructionOrder(b, placementOrder) {
			continue
		}
		return b, nil
	}
	return Board{}, fmt.Errorf("gen: could not generate a valid %d×%d board", w, h)
}

// tryPlaceOnePath tries candidate lengths (and fallback R) to place one path; returns true on success.
func tryPlaceOnePath(w, h int, occ []bool, grid []rune, emptyCount *int, rng *rand.Rand, placementOrder *[]placedComponent) bool {
	R := *emptyCount
	placed := false
	for _, L := range candidateLengths(w, h, R, rng) {
		path, ok := tryGrowPath(w, h, occ, L, rng)
		if !ok {
			continue
		}
		// Leaving exactly one empty cell makes completion impossible (paths need length >= 2).
		if R-len(path) == 1 {
			continue
		}
		if err := paintPath(grid, w, path); err != nil {
			continue
		}
		for _, p := range path {
			occ[p.y*w+p.x] = true
		}
		*emptyCount -= len(path)
		*placementOrder = append(*placementOrder, placedComponent{path: append([]point(nil), path...)})
		return true
	}
	if !placed && R >= 2 {
		path, ok := tryGrowPath(w, h, occ, R, rng)
		if ok && R-len(path) != 1 {
			if err := paintPath(grid, w, path); err == nil {
				for _, p := range path {
					occ[p.y*w+p.x] = true
				}
				*emptyCount -= len(path)
				*placementOrder = append(*placementOrder, placedComponent{path: append([]point(nil), path...)})
				return true
			}
		}
	}
	return false
}

// verifyReverseConstructionOrder checks that firing heads in reverse placement order
// (last placed path first) clears the board — the generator's intended solution.
func verifyReverseConstructionOrder(b Board, placementOrder []placedComponent) bool {
	g := NewGame(b, 1<<20, "")
	for i := len(placementOrder) - 1; i >= 0; i-- {
		path := placementOrder[i].path
		if len(path) < 2 {
			return false
		}
		hx, hy := path[0].x, path[0].y
		if !RayEscapes(g.Board, hx, hy) {
			return false
		}
		if TryFire(g, hx, hy) != FireCleared {
			return false
		}
	}
	return g.Won()
}

type point struct {
	x, y int
}

func candidateLengths(w, h, R int, rng *rand.Rand) []int {
	if R <= 2 {
		return []int{R}
	}
	n := max(w, h)
	Lmin := max(2, min(4, n))
	Lmax := min(R, max(2*n, 6))
	if Lmax < Lmin {
		Lmax = Lmin
	}
	var cand []int
	for L := Lmin; L <= Lmax && L <= R; L++ {
		rem := R - L
		if rem == 1 {
			continue
		}
		cand = append(cand, L)
	}
	if len(cand) == 0 {
		return []int{R}
	}
	rng.Shuffle(len(cand), func(i, j int) { cand[i], cand[j] = cand[j], cand[i] })
	return cand
}

func rayHitsOccupied(hx, hy int, fire Direction, occ []bool, w, h int) bool {
	dx, dy := Delta(fire)
	for x, y := hx+dx, hy+dy; x >= 0 && x < w && y >= 0 && y < h; x, y = x+dx, y+dy {
		if occ[y*w+x] {
			return true
		}
	}
	return false
}

// headRayClear returns true iff every in-bounds cell along the ray from (hx,hy) in
// direction fire is false in block (head cell itself is not on the ray).
func headRayClear(hx, hy int, fire Direction, block []bool, w, h int) bool {
	dx, dy := Delta(fire)
	for x, y := hx+dx, hy+dy; x >= 0 && x < w && y >= 0 && y < h; x, y = x+dx, y+dy {
		if block[y*w+x] {
			return false
		}
	}
	return true
}

// emptyNeighborCount counts orthogonal empty neighbors of (x,y) excluding prev and blocked cells.
func emptyNeighborCount(x, y, prevx, prevy int, occ, inPath []bool, w, h int) int {
	n := 0
	for _, nd := range []Direction{North, South, East, West} {
		dx, dy := Delta(nd)
		nx, ny := x+dx, y+dy
		if nx < 0 || nx >= w || ny < 0 || ny >= h {
			continue
		}
		if nx == prevx && ny == prevy {
			continue
		}
		if occ[ny*w+nx] || inPath[ny*w+nx] {
			continue
		}
		n++
	}
	return n
}

func tryGrowPath(w, h int, occ []bool, want int, rng *rand.Rand) ([]point, bool) {
	wh := w * h
	empty := make([]point, 0, wh)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !occ[y*w+x] {
				empty = append(empty, point{x, y})
			}
		}
	}
	if want > len(empty) || want < 2 {
		return nil, false
	}
	for attempt := 0; attempt < maxGrowAttempts; attempt++ {
		H := empty[rng.IntN(len(empty))]
		dirs := []Direction{North, South, East, West}
		rng.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
		for _, d := range dirs {
			dx, dy := Delta(d)
			bx, by := H.x+dx, H.y+dy
			if bx < 0 || bx >= w || by < 0 || by >= h {
				continue
			}
			if occ[by*w+bx] {
				continue
			}
			fire := oppositeDirGen(d)
			if rayHitsOccupied(H.x, H.y, fire, occ, w, h) {
				continue
			}
			path := []point{H, {bx, by}}
			inPath := make([]bool, wh)
			inPath[H.y*w+H.x] = true
			inPath[by*w+bx] = true
			for len(path) < want {
				tail := path[len(path)-1]
				prev := path[len(path)-2]
				var cands []point
				bestScore := -1
				for _, nd := range []Direction{North, South, East, West} {
					ndx, ndy := Delta(nd)
					nx, ny := tail.x+ndx, tail.y+ndy
					if nx < 0 || nx >= w || ny < 0 || ny >= h {
						continue
					}
					if nx == prev.x && ny == prev.y {
						continue
					}
					if occ[ny*w+nx] || inPath[ny*w+nx] {
						continue
					}
					sc := emptyNeighborCount(nx, ny, tail.x, tail.y, occ, inPath, w, h)
					if sc > bestScore {
						bestScore = sc
						cands = cands[:0]
						cands = append(cands, point{nx, ny})
					} else if sc == bestScore {
						cands = append(cands, point{nx, ny})
					}
				}
				if len(cands) == 0 {
					break
				}
				// Prefer extensions with more empty escape routes for the tail; tie-break at random.
				nxt := cands[rng.IntN(len(cands))]
				path = append(path, nxt)
				inPath[nxt.y*w+nxt.x] = true
			}
			if len(path) == want {
				fire := oppositeDirGen(d)
				block := make([]bool, wh)
				for i := range occ {
					if occ[i] {
						block[i] = true
					}
				}
				for _, p := range path {
					block[p.y*w+p.x] = true
				}
				if headRayClear(H.x, H.y, fire, block, w, h) {
					return path, true
				}
			}
		}
	}
	return nil, false
}

func oppositeDirGen(d Direction) Direction {
	switch d {
	case North:
		return South
	case South:
		return North
	case East:
		return West
	case West:
		return East
	default:
		return North
	}
}

func dirFromTo(fromx, fromy, tox, toy int) Direction {
	switch {
	case tox == fromx && toy == fromy-1:
		return North
	case tox == fromx && toy == fromy+1:
		return South
	case tox == fromx+1 && toy == fromy:
		return East
	case tox == fromx-1 && toy == fromy:
		return West
	default:
		return North
	}
}

func headRuneForFire(fire Direction) rune {
	switch fire {
	case North:
		return '▲'
	case South:
		return '▼'
	case East:
		return '▶'
	case West:
		return '◀'
	default:
		return '▲'
	}
}

func paintPath(grid []rune, w int, path []point) error {
	if len(path) < 2 {
		return fmt.Errorf("path too short")
	}
	hx, hy := path[0].x, path[0].y
	i0 := hy*w + hx
	if grid[i0] != 0 {
		return fmt.Errorf("cell occupied")
	}
	dBody := dirFromTo(hx, hy, path[1].x, path[1].y)
	grid[i0] = headRuneForFire(oppositeDirGen(dBody))

	for i := 1; i < len(path); i++ {
		px, py := path[i].x, path[i].y
		idx := py*w + px
		if grid[idx] != 0 {
			return fmt.Errorf("cell occupied")
		}
		dPrev := dirFromTo(px, py, path[i-1].x, path[i-1].y)
		if i < len(path)-1 {
			dNext := dirFromTo(px, py, path[i+1].x, path[i+1].y)
			grid[idx] = wireRuneTwo(dPrev, dNext)
		} else {
			grid[idx] = wireRuneOne(dPrev)
		}
	}
	return nil
}

func wireRuneOne(d Direction) rune {
	switch d {
	case North, South:
		return '│'
	case East, West:
		return '─'
	default:
		return '│'
	}
}

func wireRuneTwo(a, b Direction) rune {
	if a == oppositeDirGen(b) {
		if a == North || a == South {
			return '│'
		}
		return '─'
	}
	set := map[Direction]bool{a: true, b: true}
	switch {
	case set[North] && set[East]:
		return '└'
	case set[North] && set[West]:
		return '┘'
	case set[South] && set[East]:
		return '┌'
	case set[South] && set[West]:
		return '┐'
	default:
		return '│'
	}
}
