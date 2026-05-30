package game

import (
	"fmt"
	"math/rand/v2"
)

// growStraightChance10 is P(straight)/10 when both straight and turn tail steps
// exist while extending a body. Higher values produce longer straight runs.
const growStraightChance10 = 9

// cellCand is one valid (head-cell, fire-direction) candidate from
// enumerateAllCandidates. rayLen is the number of in-board cells along the
// open firing ray — used as one factor in the placement weight.
type cellCand struct {
	x, y   int
	fire   Direction
	rayLen int
}

// generateFullBoardReverse builds a w×h board by placing arrows in reverse firing
// order. Each placement enumerates all valid (cell, fire) candidates and samples
// one weighted by three factors: long firing ray (rayLen+1), spatial spread to
// the nearest same-direction head (minDist²+1), and quota deficit relative to
// the target ≈K/4 per direction (deficit²+1). These three multipliers
// simultaneously avoid edge-fire-out clusters, scatter same-direction heads, and
// keep global direction counts close to ⌈K/4⌉ ± 1 — all while respecting the
// constructive ray-clearance invariant that guarantees forward solvability.
func generateFullBoardReverse(w, h int, rng *rand.Rand) (Board, error) {
	if w <= 0 || h <= 0 {
		return Board{}, fmt.Errorf("gen: invalid size %d×%d", w, h)
	}
	wh := w * h
	if wh < 2 {
		return Board{}, fmt.Errorf("gen: need at least 2 cells (got %d×%d)", w, h)
	}

	n := min(w, h)
	nHeads := clampArrowCount(targetArrowCountForSide(n), wh)
	sc := newGenScratch(w, h, nHeads)
	bodyLenBound := wh / nHeads
	if bodyLenBound < 2 {
		bodyLenBound = 2
	}
	// Same-direction heads must be at Chebyshev distance ≥ minSameDirDist
	// from each other. 2 means they cannot touch (no immediate neighbours);
	// this is the visible "cluster" the user wanted to eliminate. The
	// constraint is hard inside enumerateAllCandidates — if it leaves no
	// candidates, the per-step countSlack relaxation is tried, and if that
	// also fails the whole board is restarted with a fresh sub-RNG.
	const minSameDirDist = 2
	const maxRestarts = 128
	for restart := 0; restart < maxRestarts; restart++ {
		sub := rand.New(rand.NewPCG(rng.Uint64(), rng.Uint64()))
		clear(sc.glyphAt[:wh])
		sc.headXs = sc.headXs[:0]
		sc.headYs = sc.headYs[:0]
		for i := range sc.headByDir {
			sc.headByDir[i] = sc.headByDir[i][:0]
		}
		targets := perDirTargets(nHeads, sub)

		placed := 0
		for placed < nHeads {
			if !placeArrowBalanced(w, h, bodyLenBound, targets, minSameDirDist, sub, sc) {
				break
			}
			placed++
		}
		if placed < nHeads {
			continue
		}
		b := NewBoard(w, h)
		for i, r := range sc.glyphAt[:wh] {
			b.Data[i] = Cell{R: r}
		}
		heads := sc.headsBuf[:0]
		for i := range sc.headXs {
			heads = append(heads, Point{sc.headXs[i], sc.headYs[i]})
		}
		sc.headsBuf = heads
		if !growPlayfulEnoughHeads(b, heads) {
			continue
		}
		return b, nil
	}
	return Board{}, fmt.Errorf("gen: could not build reverse board for %d×%d", w, h)
}

// perDirTargets returns the target head-count per fire direction such that the
// four counts sum to K and are within ±1 of each other. The +1 slots are
// rotated by a random offset so no direction is systematically favoured.
func perDirTargets(K int, rng *rand.Rand) [4]int {
	var t [4]int
	base := K / 4
	rem := K - base*4
	offset := rng.IntN(4)
	for i := 0; i < 4; i++ {
		t[i] = base
		if (i-offset+4)%4 < rem {
			t[i]++
		}
	}
	return t
}

// placeArrowBalanced enumerates every valid (cell, fire) candidate on the
// current sc.glyphAt, samples one with weight combining ray-length, spatial
// spread to nearest same-direction head, and an under-target deficit bias,
// then commits the head and grows the body. Two hard constraints are relaxed
// progressively when no candidate exists: (a) each direction's head count
// cannot exceed targets[fire] + countSlack, and (b) the candidate cell must
// be at Chebyshev distance ≥ minSameDirDist from every head firing in the
// same direction. minSameDirDist starts at an ideal value derived from K and
// shrinks first; countSlack only grows after the distance constraint reaches
// 1 (i.e. only forbid same-cell, which is already excluded by occupancy).
func placeArrowBalanced(w, h, bodyLenBound int, targets [4]int, minSameDirDist int, rng *rand.Rand, sc *genScratch) bool {
	// Try the hard minimum distance first; if it leaves no candidates, relax
	// the count cap by one (allowing one direction to overshoot) but keep the
	// distance constraint hard. Distance is non-negotiable: it's what
	// prevents same-direction clusters. Restart-on-failure is the safety net.
	cands := enumerateAllCandidates(w, h, sc, sc.cellCands[:0], targets, 0, minSameDirDist)
	sc.cellCands = cands
	if len(cands) == 0 {
		cands = enumerateAllCandidates(w, h, sc, sc.cellCands[:0], targets, 1, minSameDirDist)
		sc.cellCands = cands
	}
	if len(cands) == 0 {
		cands = enumerateAllCandidates(w, h, sc, sc.cellCands[:0], targets, 2, minSameDirDist)
		sc.cellCands = cands
	}
	if len(cands) == 0 {
		return false
	}
	choice := pickWeighted(cands, targets, w, h, sc, rng)
	hx, hy, fire := choice.x, choice.y, choice.fire
	bdx, bdy := Delta(oppositeDir(fire))
	ax, ay := hx+bdx, hy+bdy

	sc.glyphAt[hy*w+hx] = headRuneForFire(fire)
	sc.headXs = append(sc.headXs, hx)
	sc.headYs = append(sc.headYs, hy)
	sc.headByDir[fire] = append(sc.headByDir[fire], Point{hx, hy})
	maxLen := 1 + rng.IntN(bodyLenBound)
	growBodyReverse(hx, hy, fire, ax, ay, maxLen, w, h, rng, sc)
	return true
}

// enumerateAllCandidates collects every (cell, fire) pair that can host a
// head on the current sc.glyphAt: cell empty, body anchor empty and in-bounds,
// open ray clear, anchor's dangling port wouldn't form an accidental cross-
// path link, fire direction's count is below targets[fire] + countSlack, and
// the cell is at Chebyshev distance at least minSameDirDist from every head
// already firing in that direction. The rayLen recorded is the number of in-
// board cells walked along the open ray. Output is appended to out.
func enumerateAllCandidates(w, h int, sc *genScratch, out []cellCand, targets [4]int, countSlack, minSameDirDist int) []cellCand {
	g := sc.glyphAt[:w*h]
	var dirCap [4]int
	for i := 0; i < 4; i++ {
		dirCap[i] = targets[i] + countSlack
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if g[y*w+x] != 0 {
				continue
			}
			for _, fire := range [4]Direction{North, East, South, West} {
				if len(sc.headByDir[fire]) >= dirCap[fire] {
					continue
				}
				if minSameDirDist > 0 {
					sameDir := sc.headByDir[fire]
					tooClose := false
					for _, q := range sameDir {
						dxh := x - q.X
						if dxh < 0 {
							dxh = -dxh
						}
						dyh := y - q.Y
						if dyh < 0 {
							dyh = -dyh
						}
						d := dxh
						if dyh > d {
							d = dyh
						}
						if d < minSameDirDist {
							tooClose = true
							break
						}
					}
					if tooClose {
						continue
					}
				}
				bdx, bdy := Delta(oppositeDir(fire))
				bx, by := x+bdx, y+bdy
				if bx < 0 || bx >= w || by < 0 || by >= h {
					continue
				}
				if g[by*w+bx] != 0 {
					continue
				}
				dx, dy := Delta(fire)
				clearRay := true
				rayLen := 0
				for cx, cy := x+dx, y+dy; cx >= 0 && cx < w && cy >= 0 && cy < h; cx, cy = cx+dx, cy+dy {
					if g[cy*w+cx] != 0 {
						clearRay = false
						break
					}
					rayLen++
				}
				if !clearRay {
					continue
				}
				if wouldLinkAcrossPaths(x, y, bx, by, w, h, g) {
					continue
				}
				out = append(out, cellCand{x: x, y: y, fire: fire, rayLen: rayLen})
			}
		}
	}
	return out
}

// pickWeighted samples a candidate using three multiplicative weight factors:
//
//   - (rayLen + 1)   — prefer placements with long firing rays (heads pointing
//     into the populated board) over edge-fire-out placements
//     (rayLen == 0).
//   - (minDist² + 1) — per-fire-direction spatial spread, where minDist is the
//     Chebyshev distance to the nearest already-placed head
//     facing the same direction. Squared so the influence
//     drops quickly with distance.
//   - (deficit² + 1) — bias toward under-target directions. deficit =
//     max(0, targets[fire] - count[fire]). Once a direction
//     hits target, deficit clamps at zero (factor 1); the
//     hard cap in enumerateAllCandidates is what actually
//     prevents over-running it, but this term focuses the
//     sampler on the directions that still need filling
//     when several caps are open at once.
//
// First-placement-of-a-direction is treated as "max spread" (minDist =
// max(w,h)) so the first N/E/S/W heads are governed by rayLen and deficit.
func pickWeighted(cands []cellCand, targets [4]int, w, h int, sc *genScratch, rng *rand.Rand) cellCand {
	noSpreadDist := w
	if h > w {
		noSpreadDist = h
	}
	weight := func(c cellCand) int {
		sameDir := sc.headByDir[c.fire]
		minDist := noSpreadDist
		if len(sameDir) > 0 {
			minDist = chebyshevMinDist(c.x, c.y, sameDir)
		}
		deficit := targets[c.fire] - len(sameDir)
		if deficit < 0 {
			deficit = 0
		}
		spread := minDist * minDist
		spread *= spread // minDist^4 — quadratic penalty on closeness
		return (c.rayLen + 1) * (spread + 1) * (deficit*deficit + 1)
	}
	total := 0
	for _, c := range cands {
		total += weight(c)
	}
	target := rng.IntN(total)
	for _, c := range cands {
		target -= weight(c)
		if target < 0 {
			return c
		}
	}
	return cands[len(cands)-1]
}

// chebyshevMinDist returns the smallest L∞ distance from (x,y) to any point
// in pts. Assumes pts is non-empty.
func chebyshevMinDist(x, y int, pts []Point) int {
	best := -1
	for _, p := range pts {
		dx := x - p.X
		if dx < 0 {
			dx = -dx
		}
		dy := y - p.Y
		if dy < 0 {
			dy = -dy
		}
		d := dx
		if dy > d {
			d = dy
		}
		if best < 0 || d < best {
			best = d
		}
	}
	return best
}

// growBodyReverse extends the body from anchor (ax,ay) by random walk until no
// neighbor candidate is valid or maxLen body cells have been placed. The walk
// avoids: occupied cells, the head's firing ray (cellOnOpenRayFromHead), and
// placements whose dangling port would link to a wire on another already-placed
// path (wouldLinkAcrossPaths). maxLen counts the anchor as cell 1.
func growBodyReverse(hx, hy int, fire Direction, ax, ay, maxLen, w, h int, rng *rand.Rand, sc *genScratch) {
	g := sc.glyphAt[:w*h]
	g[ay*w+ax] = wireRuneOne(directionFromTo(ax, ay, hx, hy))
	prev := Point{hx, hy}
	tail := Point{ax, ay}
	length := 1
	for length < maxLen {
		cands := sc.stepBuf[:0]
		for _, d := range [4]Direction{North, East, South, West} {
			dx, dy := Delta(d)
			nx, ny := tail.X+dx, tail.Y+dy
			if nx < 0 || nx >= w || ny < 0 || ny >= h {
				continue
			}
			if nx == prev.X && ny == prev.Y {
				continue
			}
			if g[ny*w+nx] != 0 {
				continue
			}
			if cellOnOpenRayFromHead(hx, hy, fire, nx, ny, w, h) {
				continue
			}
			if wouldLinkAcrossPaths(tail.X, tail.Y, nx, ny, w, h, g) {
				continue
			}
			cands = append(cands, Point{nx, ny})
		}
		sc.stepBuf = cands
		if len(cands) == 0 {
			return
		}
		next := pickBiasedTailStep(prev, tail, cands, rng, growStraightChance10)
		g[tail.Y*w+tail.X] = wireRuneTwo(
			directionFromTo(tail.X, tail.Y, prev.X, prev.Y),
			directionFromTo(tail.X, tail.Y, next.X, next.Y),
		)
		g[next.Y*w+next.X] = wireRuneOne(directionFromTo(next.X, next.Y, tail.X, tail.Y))
		prev = tail
		tail = next
		length++
	}
}

// wouldLinkAcrossPaths reports whether writing a degree-1 wire at (nx,ny) that
// points back at (tx,ty) would form a mutual-consent port link with any
// already-placed cell other than (tx,ty). Heads have no nominal wire ports so
// they cannot accidentally link to a foreign wire.
func wouldLinkAcrossPaths(tx, ty, nx, ny, w, h int, g []rune) bool {
	dirToT := directionFromTo(nx, ny, tx, ty)
	newGlyph := wireRuneOne(dirToT)
	for _, d := range [4]Direction{North, East, South, West} {
		dx, dy := Delta(d)
		ax, ay := nx+dx, ny+dy
		if ax == tx && ay == ty {
			continue
		}
		if ax < 0 || ax >= w || ay < 0 || ay >= h {
			continue
		}
		other := g[ay*w+ax]
		if other == 0 {
			continue
		}
		if linked(Cell{R: newGlyph}, Cell{R: other}, d) {
			return true
		}
	}
	return false
}

// pickBiasedTailStep chooses the next cell when extending a polyline tail. When
// both a straight continuation and a turn are legal, straightChance10 out of 10
// rolls pick straight.
func pickBiasedTailStep(prev, tail Point, cands []Point, rng *rand.Rand, straightChance10 int) Point {
	if len(cands) == 1 {
		return cands[0]
	}
	incoming := directionFromTo(prev.X, prev.Y, tail.X, tail.Y)
	var straight, turn []Point
	for _, c := range cands {
		out := directionFromTo(tail.X, tail.Y, c.X, c.Y)
		if out == incoming {
			straight = append(straight, c)
		} else {
			turn = append(turn, c)
		}
	}
	if len(turn) > 0 && len(straight) > 0 {
		if rng.IntN(10) < straightChance10 {
			return straight[rng.IntN(len(straight))]
		}
		return turn[rng.IntN(len(turn))]
	}
	return cands[rng.IntN(len(cands))]
}

// cellOnOpenRayFromHead reports whether (px,py) lies on the open ray from
// (hx,hy) in direction fire — the first cell is (hx,hy)+Delta(fire), excluding
// the head itself. Matches RayEscapes ray traversal.
func cellOnOpenRayFromHead(hx, hy int, fire Direction, px, py, w, h int) bool {
	dx, dy := Delta(fire)
	for cx, cy := hx+dx, hy+dy; cx >= 0 && cx < w && cy >= 0 && cy < h; cx, cy = cx+dx, cy+dy {
		if cx == px && cy == py {
			return true
		}
	}
	return false
}
