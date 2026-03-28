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

// GenerateFullBoard fills a w×h grid with multiple arrow paths using reverse construction:
// components are placed so firing them in reverse placement order clears the board.
// Primary: randomized greedy multi-segment fill, then bounded DFS. Next: K horizontal bands,
// each a Hamiltonian snake (K≥3 heads). Fallback: a single snake (always succeeds for wh≥2).
// Accepted boards pass acceptPlayful when applicable (single-snake and tiny grids skip strict checks).
func GenerateFullBoard(w, h int, rng *rand.Rand) (Board, error) {
	wh := w * h
	if w <= 0 || h <= 0 {
		return Board{}, fmt.Errorf("gen: invalid size %d×%d", w, h)
	}
	if wh < 2 {
		return Board{}, fmt.Errorf("gen: need at least 2 cells (got %d×%d)", w, h)
	}

	// 1) Greedy random multi-segment partition.
	greedyTries := 6000 + 100*wh
	if greedyTries > 45000 {
		greedyTries = 45000
	}
	var placementOrder []placedComponent
	for attempt := 0; attempt < greedyTries; attempt++ {
		var ok bool
		placementOrder, ok = tryGreedyPartition(w, h, wh, rng)
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

	// 2) K horizontal bands (K≥3 snakes), bottom band placed first — fast structured fallback.
	if wh >= 12 && h >= 4 {
		bandTries := 120 + wh
		if bandTries > 700 {
			bandTries = 700
		}
		for attempt := 0; attempt < bandTries; attempt++ {
			order, ok := tryKHorizontalBands(w, h, rng)
			if !ok {
				continue
			}
			b, err := buildAndVerifyBoard(w, h, wh, order)
			if err != nil {
				continue
			}
			if acceptPlayful(b, order, w, h) {
				return b, nil
			}
		}
	}

	// 3) Single-snake fallback (one component; playfulness skips strict rules).
	path := generateSnakePathOriented(w, h, rng)
	if path == nil {
		return Board{}, fmt.Errorf("gen: could not build board for %d×%d", w, h)
	}
	placementOrder = []placedComponent{{path: append([]point(nil), path...)}}
	return buildBoardFromPlacement(w, h, wh, placementOrder)
}

func buildBoardFromPlacement(w, h, wh int, placementOrder []placedComponent) (Board, error) {
	return buildAndVerifyBoard(w, h, wh, placementOrder)
}

// tryKHorizontalBands partitions rows into K≥3 horizontal strips; each strip is one snake.
// Placement order is bottom strip first, top strip last (reverse play clears top→bottom).
func tryKHorizontalBands(w, h int, rng *rand.Rand) ([]placedComponent, bool) {
	if h < 4 {
		return nil, false
	}
	kMax := minInt(8, h-1)
	if kMax < 3 {
		return nil, false
	}
	K := 3 + rng.IntN(kMax-3+1)
	heights := randomBandHeights(h, K, w, rng)
	if heights == nil {
		return nil, false
	}
	var order []placedComponent
	yOff := 0
	for _, hi := range heights {
		path := generateSnakePathOriented(w, hi, rng)
		if path == nil {
			return nil, false
		}
		shifted := make([]point, len(path))
		for j := range path {
			shifted[j] = point{path[j].x, path[j].y + yOff}
		}
		order = append(order, placedComponent{path: append([]point(nil), shifted...)})
		yOff += hi
	}
	if yOff != h {
		return nil, false
	}
	return order, true
}

// randomBandHeights splits h rows into K positive heights summing to h; each band has at least 2 cells
// (so a nontrivial arrow path exists), except when w≥2 a height-1 band still has ≥2 cells across the row.
func randomBandHeights(h, K, w int, rng *rand.Rand) []int {
	minH := 1
	if w == 1 {
		minH = 2
	}
	if h < K*minH {
		return nil
	}
	heights := make([]int, K)
	for i := range heights {
		heights[i] = minH
	}
	rem := h - K*minH
	for rem > 0 {
		heights[rng.IntN(K)]++
		rem--
	}
	for _, hi := range heights {
		if w*hi < 2 {
			return nil
		}
	}
	return heights
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

// tryGreedyPartition fills the board with one forward pass of random segments.
func tryGreedyPartition(w, h, wh int, rng *rand.Rand) ([]placedComponent, bool) {
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
// Sampling is skewed toward mid-length segments and occasionally the full remainder.
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
		var cands []int
		var weights []int
		mid := (2 + hi) / 2
		for t := 2; t <= hi; t++ {
			if rem-t == 1 {
				continue
			}
			cands = append(cands, t)
			wt := 3 + absInt(t-mid)
			if t == rem {
				wt += 2 + hi/4
			}
			weights = append(weights, wt)
		}
		if len(cands) == 0 {
			return rem
		}
		total := 0
		for _, w := range weights {
			total += w
		}
		r := rng.IntN(total)
		s := 0
		for i, w := range weights {
			s += w
			if r < s {
				return cands[i]
			}
		}
		return cands[len(cands)-1]
	}
}

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

// pickBiasedTailStep prefers 90° turns over straight continuation when both exist (more varied shapes).
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

// --- Single-snake helpers (fallback and sub-rectangles in split generation) ---

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
// on the first step (required when every cell of that component is occupied).
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
	return nil
}

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
