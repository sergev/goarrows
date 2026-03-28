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

// GenerateFullBoard fills a w×h grid with arrow polylines using inverse construction: paths are
// placed in order so firing them in reverse placement order clears the board. A shuffled cell queue
// drives candidate heads; each path has length ≥ 2 and satisfies ray-clear from the head. Retries
// until acceptPlayful passes or the attempt budget is exhausted (no snake or greedy fallback).
func GenerateFullBoard(w, h int, rng *rand.Rand) (Board, error) {
	wh := w * h
	if w <= 0 || h <= 0 {
		return Board{}, fmt.Errorf("gen: invalid size %d×%d", w, h)
	}
	if wh < 2 {
		return Board{}, fmt.Errorf("gen: need at least 2 cells (got %d×%d)", w, h)
	}

	// Bounded outer attempts; each inner tryInversePolylinePartition fills until success or dead end.
	maxTries := 12000 + 120*wh
	if maxTries > 80000 {
		maxTries = 80000
	}
	for attempt := 0; attempt < maxTries; attempt++ {
		// Fresh RNG per attempt so retries explore different partitions (reusing the same rng would repeat identical fills).
		r := rand.New(rand.NewPCG(rng.Uint64(), rng.Uint64()))
		placementOrder, ok := tryInversePolylinePartition(w, h, wh, r)
		if !ok {
			continue
		}
		b, err := buildAndVerifyBoard(w, h, wh, placementOrder)
		if err != nil {
			continue
		}
		if acceptPlayful(b, placementOrder, w, h) {
			return b, nil
		}
	}
	return Board{}, fmt.Errorf("gen: could not build board for %d×%d", w, h)
}

// tryInversePolylinePartition fills the grid under the reverse-construction invariant. Each
// segment uses tryPlaceRandomPath (shuffled free heads). Optionally places a small template
// polyline first (same idea as the old greedy generator) to reduce dead ends on medium/large boards.
func tryInversePolylinePartition(w, h, wh int, rng *rand.Rand) ([]placedComponent, bool) {
	occupied := make([]bool, wh)
	order := make([]placedComponent, 0, wh/2+2)
	if wh >= 20 && rng.IntN(14) == 0 {
		if tpl := tryPlaceRandomTemplate(w, h, occupied, rng); tpl != nil {
			for _, p := range tpl {
				occupied[p.y*w+p.x] = true
			}
			order = append(order, placedComponent{path: append([]point(nil), tpl...)})
		}
	}

	for countFilled(occupied) < wh {
		rem := wh - countFilled(occupied)
		if rem == 1 {
			return nil, false
		}
		tl := pickTargetLength(rem, rng)
		path := tryPlaceRandomPath(w, h, occupied, rng, tl)
		if path == nil {
			return nil, false
		}
		for _, p := range path {
			occupied[p.y*w+p.x] = true
		}
		order = append(order, placedComponent{path: append([]point(nil), path...)})
	}
	return order, true
}

// tryPlaceRandomTemplate places a small fixed zig-zag at a random offset when it satisfies ray-clear rules.
func tryPlaceRandomTemplate(w, h int, occupied []bool, rng *rand.Rand) []point {
	templates := [][]point{
		{{0, 0}, {1, 0}, {1, 1}, {2, 1}},
		{{0, 0}, {0, 1}, {1, 1}, {2, 1}},
	}
	base := templates[rng.IntN(len(templates))]
	maxX, maxY := 0, 0
	for _, p := range base {
		if p.x > maxX {
			maxX = p.x
		}
		if p.y > maxY {
			maxY = p.y
		}
	}
	if w <= maxX || h <= maxY {
		return nil
	}
	for t := 0; t < 45; t++ {
		ox := rng.IntN(w - maxX)
		oy := rng.IntN(h - maxY)
		path := make([]point, len(base))
		for i := range base {
			path[i] = point{base[i].x + ox, base[i].y + oy}
		}
		if templatePathValid(w, h, occupied, path) {
			return path
		}
	}
	return nil
}

func templatePathValid(w, h int, occupied []bool, path []point) bool {
	for i := range path {
		p := path[i]
		if p.x < 0 || p.y < 0 || p.x >= w || p.y >= h {
			return false
		}
		if occupied[p.y*w+p.x] {
			return false
		}
	}
	for i := 0; i < len(path)-1; i++ {
		d := absInt(path[i].x-path[i+1].x) + absInt(path[i].y-path[i+1].y)
		if d != 1 {
			return false
		}
	}
	hx, hy := path[0].x, path[0].y
	dBody := dirFromTo(hx, hy, path[1].x, path[1].y)
	fire := oppositeDirGen(dBody)
	if !rayClearFromHead(hx, hy, fire, occupied, w, h) {
		return false
	}
	return true
}

// tryPlaceRandomPath chooses a random free cell as head and grows a ray-clear polyline (length ≥ 2).
func tryPlaceRandomPath(w, h int, occupied []bool, rng *rand.Rand, targetLen int) []point {
	if targetLen < 2 {
		targetLen = 2
	}
	wh := w * h
	free := make([]point, 0, wh)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !occupied[y*w+x] {
				free = append(free, point{x, y})
			}
		}
	}
	rng.Shuffle(len(free), func(i, j int) {
		free[i], free[j] = free[j], free[i]
	})

	dirs := []Direction{North, East, South, West}

	for _, head := range free {
		rng.Shuffle(len(dirs), func(i, j int) {
			dirs[i], dirs[j] = dirs[j], dirs[i]
		})
		for _, dBody := range dirs {
			dx, dy := Delta(dBody)
			nx, ny := head.x+dx, head.y+dy
			if nx < 0 || nx >= w || ny < 0 || ny >= h {
				continue
			}
			if occupied[ny*w+nx] {
				continue
			}
			fire := oppositeDirGen(dBody)
			if !rayClearFromHead(head.x, head.y, fire, occupied, w, h) {
				continue
			}
			path := []point{head, {nx, ny}}
			pathSet := map[point]struct{}{head: {}, {nx, ny}: {}}

			for len(path) < targetLen {
				tail := path[len(path)-1]
				prev := path[len(path)-2]
				cands := neighborPoints(tail, prev, w, h, occupied, pathSet)
				if len(cands) == 0 {
					break
				}
				next := pickBiasedTailStep(prev, tail, cands, rng)
				path = append(path, next)
				pathSet[next] = struct{}{}
			}
			if len(path) >= 2 {
				return path
			}
		}
	}
	return nil
}

func buildAndVerifyBoard(w, h, wh int, placementOrder []placedComponent) (Board, error) {
	grid := make([]rune, wh)
	for i := range placementOrder {
		if err := paintPath(grid, w, placementOrder[i].path); err != nil {
			return Board{}, err
		}
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
	if !verifyReverseConstructionOrder(b, placementOrder) {
		return Board{}, fmt.Errorf("gen: reverse-order verification failed")
	}
	return b, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func countFilled(occupied []bool) int {
	n := 0
	for _, v := range occupied {
		if v {
			n++
		}
	}
	return n
}

// pickTargetLength chooses how many cells the next component should try to cover.
// It never picks a length that would leave exactly one cell empty (no valid path of length ≥2).
func pickTargetLength(rem int, rng *rand.Rand) int {
	switch rem {
	case 0, 1:
		return rem
	case 2, 3:
		return rem
	default:
		hi := rem
		if hi > 12 {
			hi = 12
		}
		var cands [12]int
		var weights [12]int
		mid := (2 + hi) / 2
		n := 0
		for t := 2; t <= hi; t++ {
			if rem-t == 1 {
				continue
			}
			cands[n] = t
			wt := 3 + absInt(t-mid)
			if t == rem {
				wt += 2 + hi/4
			}
			weights[n] = wt
			n++
		}
		if n == 0 {
			return rem
		}
		total := 0
		for i := 0; i < n; i++ {
			total += weights[i]
		}
		r := rng.IntN(total)
		s := 0
		for i := 0; i < n; i++ {
			s += weights[i]
			if r < s {
				return cands[i]
			}
		}
		return cands[n-1]
	}
}

func rayClearFromHead(hx, hy int, fire Direction, occupied []bool, w, h int) bool {
	dx, dy := Delta(fire)
	for cx, cy := hx+dx, hy+dy; cx >= 0 && cx < w && cy >= 0 && cy < h; cx, cy = cx+dx, cy+dy {
		if occupied[cy*w+cx] {
			return false
		}
	}
	return true
}

func neighborPoints(tail, prev point, w, h int, occupied []bool, pathSet map[point]struct{}) []point {
	var out []point
	for _, d := range []Direction{North, East, South, West} {
		dx, dy := Delta(d)
		nx, ny := tail.x+dx, tail.y+dy
		if nx < 0 || nx >= w || ny < 0 || ny >= h {
			continue
		}
		np := point{nx, ny}
		if np == prev {
			continue
		}
		if occupied[ny*w+nx] {
			continue
		}
		if _, ok := pathSet[np]; ok {
			continue
		}
		out = append(out, np)
	}
	return out
}

func pickBiasedTailStep(prev, tail point, cands []point, rng *rand.Rand) point {
	if len(cands) == 1 {
		return cands[0]
	}
	incoming := dirFromTo(prev.x, prev.y, tail.x, tail.y)
	var straight, turn []point
	for _, c := range cands {
		out := dirFromTo(tail.x, tail.y, c.x, c.y)
		if out == incoming {
			straight = append(straight, c)
		} else {
			turn = append(turn, c)
		}
	}
	if len(turn) > 0 && len(straight) > 0 {
		if rng.IntN(10) < 6 {
			return turn[rng.IntN(len(turn))]
		}
		return straight[rng.IntN(len(straight))]
	}
	return cands[rng.IntN(len(cands))]
}

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
