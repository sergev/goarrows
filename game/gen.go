package game

import (
	"fmt"
	"math/rand/v2"
)

// placedComponent records one polyline placed during generation (head at path[0]).
type placedComponent struct {
	path []point
}

type point struct {
	x, y int
}

// GenerateFullBoard fills a w×h grid with a single snake path covering every cell.
// Variety comes from RNG-driven flips / transpose (square) / reversal. The path is
// oriented so the head can fire off the edge in one step (required on a full board).
func GenerateFullBoard(w, h int, rng *rand.Rand) (Board, error) {
	wh := w * h
	if w <= 0 || h <= 0 {
		return Board{}, fmt.Errorf("gen: invalid size %d×%d", w, h)
	}
	if wh < 2 {
		return Board{}, fmt.Errorf("gen: need at least 2 cells (got %d×%d)", w, h)
	}

	path := generateSnakePathOriented(w, h, rng)
	if path == nil {
		return Board{}, fmt.Errorf("gen: could not build snake for %d×%d board", w, h)
	}

	grid := make([]rune, wh)
	if err := paintPath(grid, w, path); err != nil {
		return Board{}, err
	}

	b := NewBoard(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			b.Set(x, y, Cell{R: grid[y*w+x]})
		}
	}
	if err := ValidateBoard(b); err != nil {
		return Board{}, err
	}
	placementOrder := []placedComponent{{path: append([]point(nil), path...)}}
	if !verifyReverseConstructionOrder(b, placementOrder) {
		return Board{}, fmt.Errorf("gen: internal validation failed for %d×%d board", w, h)
	}
	return b, nil
}

// buildSnakePathRowMajor visits every cell in row-major order with alternating direction.
func buildSnakePathRowMajor(w, h int) []point {
	out := make([]point, 0, w*h)
	for y := 0; y < h; y++ {
		if y%2 == 0 {
			for x := 0; x < w; x++ {
				out = append(out, point{x, y})
			}
		} else {
			for x := w - 1; x >= 0; x-- {
				out = append(out, point{x, y})
			}
		}
	}
	return out
}

// buildSnakePathColMajor visits every cell in column-major order with alternating direction.
func buildSnakePathColMajor(w, h int) []point {
	out := make([]point, 0, w*h)
	for x := 0; x < w; x++ {
		if x%2 == 0 {
			for y := 0; y < h; y++ {
				out = append(out, point{x, y})
			}
		} else {
			for y := h - 1; y >= 0; y-- {
				out = append(out, point{x, y})
			}
		}
	}
	return out
}

func applyRandomPathTransforms(path []point, w, h int, rng *rand.Rand) []point {
	out := make([]point, len(path))
	copy(out, path)
	flipX := rng.IntN(2) == 1
	flipY := rng.IntN(2) == 1
	transpose := w == h && rng.IntN(2) == 1
	if rng.IntN(2) == 1 {
		reversePointsInPlace(out)
	}
	for i := range out {
		x, y := out[i].x, out[i].y
		if flipX {
			x = w - 1 - x
		}
		if flipY {
			y = h - 1 - y
		}
		if transpose {
			x, y = y, x
		}
		out[i] = point{x, y}
	}
	return out
}

func reversePointsInPlace(p []point) {
	for i, j := 0, len(p)-1; i < j; i, j = i+1, j-1 {
		p[i], p[j] = p[j], p[i]
	}
}

// orientPathForHeadEscape ensures path[0] is the head and the fire ray leaves the board
// on the first step (required when every cell is occupied).
func orientPathForHeadEscape(path []point, w, h int) []point {
	if len(path) < 2 {
		return nil
	}
	if headRayExitsBoard(path, w, h) {
		return path
	}
	path2 := append([]point(nil), path...)
	reversePointsInPlace(path2)
	if headRayExitsBoard(path2, w, h) {
		return path2
	}
	// Alternate Hamiltonian path; transforms were applied — rebuild base alt and retry once.
	return nil
}

// generateSnakePathOriented tries row-major then column-major snake patterns with RNG transforms.
func generateSnakePathOriented(w, h int, rng *rand.Rand) []point {
	candidates := []func(int, int) []point{
		buildSnakePathRowMajor,
		buildSnakePathColMajor,
	}
	for _, build := range candidates {
		path := build(w, h)
		path = applyRandomPathTransforms(path, w, h, rng)
		if oriented := orientPathForHeadEscape(path, w, h); oriented != nil {
			return oriented
		}
	}
	return nil
}

func headRayExitsBoard(path []point, w, h int) bool {
	if len(path) < 2 {
		return false
	}
	hx, hy := path[0].x, path[0].y
	dBody := dirFromTo(hx, hy, path[1].x, path[1].y)
	fire := oppositeDirGen(dBody)
	dx, dy := Delta(fire)
	cx, cy := hx+dx, hy+dy
	return cx < 0 || cx >= w || cy < 0 || cy >= h
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
