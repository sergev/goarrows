package game

import (
	"errors"
	"math/rand/v2"
)

// GenGrow is the only supported procedural generation algorithm.
const GenGrow = "grow"

// growStraightChance10 is P(straight)/10 when both straight and turn tail steps exist
// during grow algorithm extensions in tryGrowPartition.
const growStraightChance10 = 9

// GenerateBoard fills a w×h grid with the grow procedural algorithm.
func GenerateBoard(w, h int, rng *rand.Rand) (Board, error) {
	return generateFullBoardGrow(w, h, rng)
}

// GenerateFullBoard is kept as a compatibility alias for tests/callers.
func GenerateFullBoard(w, h int, rng *rand.Rand) (Board, error) {
	return GenerateBoard(w, h, rng)
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

// cellOnOpenRayFromHead reports whether (px, py) lies on the open ray from (hx, hy) in
// direction fire: the first cell is (hx, hy)+Delta(fire), excluding the head cell itself.
// Matches RayEscapes ray traversal.
func cellOnOpenRayFromHead(hx, hy int, fire Direction, px, py, w, h int) bool {
	dx, dy := Delta(fire)
	for cx, cy := hx+dx, hy+dy; cx >= 0 && cx < w && cy >= 0 && cy < h; cx, cy = cx+dx, cy+dy {
		if cx == px && cy == py {
			return true
		}
	}
	return false
}

type point struct {
	x, y int
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

// pickBiasedTailStep chooses the next cell when extending a polyline tail. When both a straight
// continuation and a turn are legal, straightChance10 out of 10 rolls pick straight.
func pickBiasedTailStep(prev, tail point, cands []point, rng *rand.Rand, straightChance10 int) point {
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
		if rng.IntN(10) < straightChance10 {
			return straight[rng.IntN(len(straight))]
		}
		return turn[rng.IntN(len(turn))]
	}
	return cands[rng.IntN(len(cands))]
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
		return errors.New("path too short")
	}
	hx, hy := path[0].x, path[0].y
	i0 := hy*w + hx
	if grid[i0] != 0 {
		return errors.New("cell occupied")
	}
	dBody := dirFromTo(hx, hy, path[1].x, path[1].y)
	grid[i0] = headRuneForFire(oppositeDirGen(dBody))

	for i := 1; i < len(path); i++ {
		px, py := path[i].x, path[i].y
		idx := py*w + px
		if grid[idx] != 0 {
			return errors.New("cell occupied")
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

