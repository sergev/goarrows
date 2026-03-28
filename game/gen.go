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
// Primary strategy: partition the board into two rectangles, each filled with a random
// Hamiltonian snake (variety via flips / transpose / reversal), placed in an order that
// preserves the ray-clear invariant. Secondary: randomized greedy multi-segment fill.
// Fallback: a single snake covering the whole board (always succeeds for wh≥2).
// Accepted boards also pass acceptPlayful when possible (single-snake and tiny grids skip strict playfulness).
func GenerateFullBoard(w, h int, rng *rand.Rand) (Board, error) {
	wh := w * h
	if w <= 0 || h <= 0 {
		return Board{}, fmt.Errorf("gen: invalid size %d×%d", w, h)
	}
	if wh < 2 {
		return Board{}, fmt.Errorf("gen: need at least 2 cells (got %d×%d)", w, h)
	}

	// 1) Split-board: two snakes — resample until playfulness accepts (when applicable).
	if wh >= 8 {
		splitTries := 80 + wh/2
		if splitTries > 400 {
			splitTries = 400
		}
		for attempt := 0; attempt < splitTries; attempt++ {
			order, ok := tryGenerateSplitTwoSnakes(w, h, rng)
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

	// 2) Greedy random multi-segment partition.
	greedyTries := 12000 + 200*wh
	if greedyTries > 90000 {
		greedyTries = 90000
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

	// 3) Bounded DFS backtracking.
	dfsRestarts := 800 + 30*wh
	if dfsRestarts > 8000 {
		dfsRestarts = 8000
	}
	for attempt := 0; attempt < dfsRestarts; attempt++ {
		var ok bool
		placementOrder, ok = partitionBoardDFS(w, h, wh, rng)
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

	// 4) Single-snake fallback (one component; playfulness skipped inside acceptPlayful).
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

// tryGenerateSplitTwoSnakes builds two disjoint Hamiltonian snakes in a 2-rectangle partition.
// Jagged or Voronoi-style region boundaries would require Hamiltonian paths on arbitrary polyominoes;
// those are not implemented here—variety instead comes from biased cut lines and repeated random splits.
func tryGenerateSplitTwoSnakes(w, h int, rng *rand.Rand) ([]placedComponent, bool) {
	wh := w * h
	if wh < 8 {
		return nil, false
	}
	type splitFn func(int, int, *rand.Rand) ([]placedComponent, bool)
	var fns []splitFn
	if h >= 3 {
		fns = append(fns, splitHorizontalTwoSnakes)
	}
	if w >= 3 {
		fns = append(fns, splitVerticalTwoSnakes)
	}
	if len(fns) == 0 {
		return nil, false
	}
	rng.Shuffle(len(fns), func(i, j int) {
		fns[i], fns[j] = fns[j], fns[i]
	})
	for _, fn := range fns {
		// Several random split positions per orientation (biased k inside each attempt).
		for rep := 0; rep < 5; rep++ {
			if o, ok := fn(w, h, rng); ok {
				return o, true
			}
		}
	}
	return nil, false
}

func splitHorizontalTwoSnakes(w, h int, rng *rand.Rand) ([]placedComponent, bool) {
	if h < 3 {
		return nil, false
	}
	k := pickBiasedSplitK(h-2, rng)
	if k*w < 2 || (h-k)*w < 2 {
		return nil, false
	}
	pathTop := generateSnakePathOriented(w, k, rng)
	pathBot := generateSnakePathOriented(w, h-k, rng)
	if pathTop == nil || pathBot == nil {
		return nil, false
	}
	bot := make([]point, len(pathBot))
	for i := range pathBot {
		bot[i] = point{pathBot[i].x, pathBot[i].y + k}
	}
	// Place bottom first, then top (reverse play: top fires first, then bottom).
	return []placedComponent{
		{path: bot},
		{path: append([]point(nil), pathTop...)},
	}, true
}

func splitVerticalTwoSnakes(w, h int, rng *rand.Rand) ([]placedComponent, bool) {
	if w < 3 {
		return nil, false
	}
	k := pickBiasedSplitK(w-2, rng)
	if k*h < 2 || (w-k)*h < 2 {
		return nil, false
	}
	pathLeft := generateSnakePathOriented(k, h, rng)
	pathRight := generateSnakePathOriented(w-k, h, rng)
	if pathLeft == nil || pathRight == nil {
		return nil, false
	}
	right := make([]point, len(pathRight))
	for i := range pathRight {
		right[i] = point{pathRight[i].x + k, pathRight[i].y}
	}
	// Place right first, then left.
	return []placedComponent{
		{path: right},
		{path: append([]point(nil), pathLeft...)},
	}, true
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

// partitionBoardDFS finds a full tiling by reverse-placement using backtracking.
func partitionBoardDFS(w, h, wh int, rng *rand.Rand) ([]placedComponent, bool) {
	occupied := make([]bool, wh)
	order := make([]placedComponent, 0, wh/2+2)
	var calls int
	const maxCalls = 120_000

	var dfs func() bool
	dfs = func() bool {
		calls++
		if calls > maxCalls {
			return false
		}
		if countFilled(occupied) == wh {
			return true
		}
		rem := wh - countFilled(occupied)
		lengths := candidateSegmentLengths(rem, rng)
		trialsPerLen := 28
		if rem <= 18 {
			trialsPerLen = 45
		}
		for _, tl := range lengths {
			for trial := 0; trial < trialsPerLen; trial++ {
				path := tryPlaceRandomPath(w, h, occupied, rng, tl)
				if path == nil {
					continue
				}
				for _, p := range path {
					occupied[p.y*w+p.x] = true
				}
				order = append(order, placedComponent{path: append([]point(nil), path...)})
				if dfs() {
					return true
				}
				order = order[:len(order)-1]
				for _, p := range path {
					occupied[p.y*w+p.x] = false
				}
			}
		}
		return false
	}
	if dfs() {
		return order, true
	}
	return nil, false
}

// candidateSegmentLengths returns an ordered list of segment sizes to try for the next piece.
func candidateSegmentLengths(rem int, rng *rand.Rand) []int {
	if rem <= 3 {
		return []int{rem}
	}
	primary := pickTargetLength(rem, rng)
	seen := map[int]bool{primary: true}
	out := []int{primary}
	add := func(t int) {
		if t < 2 || t > rem {
			return
		}
		if rem-t == 1 {
			return
		}
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	add(rem)
	add(2)
	if rem > 4 {
		add(rem - 2)
		add(minInt(rem, 6))
		add(minInt(rem, 8))
		add(minInt(rem, 10))
	}
	// Randomize follow-up order (keep primary first).
	if len(out) > 1 {
		rng.Shuffle(len(out)-1, func(i, j int) {
			out[i+1], out[j+1] = out[j+1], out[i+1]
		})
	}
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// pickBiasedSplitK returns an integer in [1, span] matching the old 1+rand.IntN(span) distribution
// but resampling a few times to avoid a perfectly centered split when span is large.
func pickBiasedSplitK(span int, rng *rand.Rand) int {
	if span < 1 {
		return 1
	}
	lo, hi := 1, span
	mid := (lo + hi) / 2
	for t := 0; t < 20; t++ {
		k := lo + rng.IntN(hi-lo+1)
		if span >= 6 && absInt(k-mid) <= 1 {
			continue
		}
		return k
	}
	return lo + rng.IntN(hi-lo+1)
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
