package game

import (
	"fmt"
	"math/rand/v2"
)

// growStraightChance10 is P(straight)/10 when both straight and turn tail steps
// exist while extending a body. Higher values produce longer straight runs.
const growStraightChance10 = 9

// rayCand is one valid (head, fire) placement found by enumerateClearRayCandidates.
// rayLen is the number of in-board cells the open firing ray walks before leaving
// the board — used as the sampling weight (longer rays preferred) so heads don't
// all face the nearest edge.
type rayCand struct {
	x, y   int
	fire   Direction
	rayLen int
}

// generateFullBoardReverse builds a w×h board by placing arrows in reverse firing
// order. Each placement requires that the new head's open firing ray is clear on
// the current partial board, so forward play clears every arrow by construction —
// no ValidatePartialBoard / VerifySolvableFast rejection is needed in production.
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
	// bodyLenBound caps the per-arrow body length to keep arrows competing for
	// space rather than letting the first-placed arrow consume the grid. The
	// random walk still varies length per-arrow within [1, bodyLenBound], so
	// shapes stay diverse. 2*wh/nHeads centers the distribution around the
	// average cells-per-arrow needed for ~full coverage.
	bodyLenBound := 2 * wh / nHeads
	if bodyLenBound < 2 {
		bodyLenBound = 2
	}
	const maxRestarts = 128
	for restart := 0; restart < maxRestarts; restart++ {
		// Sub-RNG per restart so reproducibility holds: a fixed parent PCG state
		// always consumes the same number of uint64s, regardless of how many
		// restarts the inner loop performs.
		sub := rand.New(rand.NewPCG(rng.Uint64(), rng.Uint64()))
		clear(sc.glyphAt[:wh])
		sc.headXs = sc.headXs[:0]
		sc.headYs = sc.headYs[:0]
		placed := 0
		for placed < nHeads {
			if !placeArrowReverse(w, h, bodyLenBound, sub, sc) {
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

// placeArrowReverse picks an (empty cell, fire direction) with a clear ray on
// the current sc.glyphAt, grows a body away from it (capped at a random length
// in [1, bodyLenBound]), and writes both the head and body glyphs. Returns
// false when no valid placement exists.
func placeArrowReverse(w, h, bodyLenBound int, rng *rand.Rand, sc *genScratch) bool {
	cands := enumerateClearRayCandidates(w, h, sc, sc.candBuf[:0])
	sc.candBuf = cands
	if len(cands) == 0 {
		return false
	}
	choice := pickWeightedRay(cands, rng)
	hx, hy, fire := choice.x, choice.y, choice.fire

	bdx, bdy := Delta(oppositeDir(fire))
	ax, ay := hx+bdx, hy+bdy

	// Reserve the head cell before body growth so the random walk treats it as
	// occupied and never wanders back through it. Head glyphs are not wires, so
	// they cannot accidentally link with any neighbor (NominalPorts(head) == 0
	// and linkMask(head) only exposes the body-direction port, which faces the
	// anchor on this same path).
	sc.glyphAt[hy*w+hx] = headRuneForFire(fire)
	sc.headXs = append(sc.headXs, hx)
	sc.headYs = append(sc.headYs, hy)
	maxLen := 1 + rng.IntN(bodyLenBound)
	growBodyReverse(hx, hy, fire, ax, ay, maxLen, w, h, rng, sc)
	return true
}

// pickWeightedRay samples a rayCand with weight proportional to rayLen+1. This
// down-weights edge-fire-out placements (rayLen == 0) so heads don't all face
// the nearest board edge. The +1 keeps short-ray placements possible when
// long-ray ones aren't available (small or nearly-full boards).
func pickWeightedRay(cands []rayCand, rng *rand.Rand) rayCand {
	total := 0
	for _, c := range cands {
		total += c.rayLen + 1
	}
	target := rng.IntN(total)
	for _, c := range cands {
		target -= c.rayLen + 1
		if target < 0 {
			return c
		}
	}
	return cands[len(cands)-1]
}

// enumerateClearRayCandidates scans every (empty cell, fire direction) pair and
// keeps those that meet the placement preconditions on the current sc.glyphAt:
// the body anchor is empty and in-bounds, the open firing ray contains no
// occupied cells, and the anchor's dangling port (the wire face opposite the
// head) wouldn't form an accidental cross-path link with the cell beyond it.
// Output is appended to out (caller-owned).
func enumerateClearRayCandidates(w, h int, sc *genScratch, out []rayCand) []rayCand {
	g := sc.glyphAt[:w*h]
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if g[y*w+x] != 0 {
				continue
			}
			for _, fire := range [4]Direction{North, East, South, West} {
				bdx, bdy := Delta(oppositeDir(fire))
				bx, by := x+bdx, y+bdy
				if bx < 0 || bx >= w || by < 0 || by >= h {
					continue
				}
				if g[by*w+bx] != 0 {
					continue
				}
				dx, dy := Delta(fire)
				clear := true
				rayLen := 0
				for cx, cy := x+dx, y+dy; cx >= 0 && cx < w && cy >= 0 && cy < h; cx, cy = cx+dx, cy+dy {
					if g[cy*w+cx] != 0 {
						clear = false
						break
					}
					rayLen++
				}
				if !clear {
					continue
				}
				// Anchor (bx,by) will be written as wireRuneOne pointing at
				// the head (x,y). Its dangling port points further away from
				// the head — verify that face doesn't accidentally link.
				if wouldLinkAcrossPaths(x, y, bx, by, w, h, g) {
					continue
				}
				out = append(out, rayCand{x: x, y: y, fire: fire, rayLen: rayLen})
			}
		}
	}
	return out
}

// growBodyReverse extends the body from anchor (ax,ay) by random walk until no
// neighbor candidate is valid or maxLen body cells have been placed. The walk
// avoids: occupied cells, the head's firing ray (cellOnOpenRayFromHead), and
// placements whose dangling port would link to a wire on another already-placed
// path (wouldLinkAcrossPaths). maxLen counts the anchor as cell 1.
func growBodyReverse(hx, hy int, fire Direction, ax, ay, maxLen, w, h int, rng *rand.Rand, sc *genScratch) {
	g := sc.glyphAt[:w*h]
	// Anchor is the first body cell, written as a degree-1 wire pointing toward the head.
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
		// Promote tail from degree-1 to degree-2 wire (straight or corner).
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
// they cannot accidentally link to a foreign wire (the head's body-direction
// port already faces its own body, which is on its own path).
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
