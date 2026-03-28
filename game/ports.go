package game

import (
	"fmt"
)

// Port bits: which directions this glyph connects to adjacent cells (path graph).
const (
	PortN uint8 = 1 << iota
	PortE
	PortS
	PortW
)

// NominalPorts returns the connection bitmask for wire glyphs only (not heads).
// Box corners follow Unicode "light" box: ┌ opens south+east, etc.
func NominalPorts(r rune) uint8 {
	if r == 0 {
		return 0
	}
	switch r {
	case '─':
		return PortE | PortW
	case '│':
		return PortN | PortS
	case '┌':
		return PortS | PortE
	case '┐':
		return PortS | PortW
	case '└':
		return PortN | PortE
	case '┘':
		return PortN | PortW
	default:
		return 0
	}
}

// linkMask returns ports used for adjacency: wires use glyph geometry; heads
// connect only toward the body (opposite of fire) so adjacent foreign wires
// do not spuriously link across component boundaries.
func linkMask(c Cell) uint8 {
	if c.IsEmpty() {
		return 0
	}
	if c.IsHead() {
		if fire, ok := HeadFireDir(c.R); ok {
			return dirToPort(oppositeDir(fire))
		}
		return 0
	}
	return NominalPorts(c.R)
}

// HeadFireDir returns the ray direction for a head rune.
func HeadFireDir(r rune) (Direction, bool) {
	switch r {
	case '^', '▲':
		return North, true
	case 'v', 'V', '▼':
		return South, true
	case '<', '◀':
		return West, true
	case '>', '▶':
		return East, true
	default:
		return 0, false
	}
}

// IsHead reports whether c is an arrow head (fireable).
func (c Cell) IsHead() bool {
	_, ok := HeadFireDir(c.R)
	return ok
}

// linked returns true if a and b are adjacent along dirFromAToB and both
// allow that edge (mutual consent). Two heads never link (paths do not join).
func linked(a, b Cell, dirFromAToB Direction) bool {
	if a.IsEmpty() || b.IsEmpty() {
		return false
	}
	if a.IsHead() && b.IsHead() {
		return false
	}
	bitA := dirToPort(dirFromAToB)
	bitB := dirToPort(oppositeDir(dirFromAToB))
	return linkMask(a)&bitA != 0 && linkMask(b)&bitB != 0
}

// HorizontalLink reports whether (x,y) and (x+1,y) are mutually connected
// along the eastward edge (for drawing wide horizontal segments in the UI).
func HorizontalLink(b Board, x, y int) bool {
	if !b.InBounds(x, y) || !b.InBounds(x+1, y) {
		return false
	}
	return linked(b.At(x, y), b.At(x+1, y), East)
}

func dirToPort(d Direction) uint8 {
	switch d {
	case North:
		return PortN
	case East:
		return PortE
	case South:
		return PortS
	case West:
		return PortW
	default:
		return 0
	}
}

func oppositeDir(d Direction) Direction {
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
		return d
	}
}

// EffectivePorts is the intersection of nominal ports with edges that both
// cells agree on (so │ does not link south to a neighbor that has no north).
func EffectivePorts(b Board, x, y int) uint8 {
	if !b.InBounds(x, y) || b.At(x, y).IsEmpty() {
		return 0
	}
	n := linkMask(b.At(x, y))
	var m uint8
	if n&PortN != 0 && b.InBounds(x, y-1) && linked(b.At(x, y), b.At(x, y-1), North) {
		m |= PortN
	}
	if n&PortE != 0 && b.InBounds(x+1, y) && linked(b.At(x, y), b.At(x+1, y), East) {
		m |= PortE
	}
	if n&PortS != 0 && b.InBounds(x, y+1) && linked(b.At(x, y), b.At(x, y+1), South) {
		m |= PortS
	}
	if n&PortW != 0 && b.InBounds(x-1, y) && linked(b.At(x, y), b.At(x-1, y), West) {
		m |= PortW
	}
	return m
}

// ValidateBoard checks full coverage, mutual links, path shape, and one head per component.
func ValidateBoard(b Board) error {
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			c := b.At(x, y)
			if c.IsEmpty() {
				return fmt.Errorf("cell (%d,%d): empty cell not allowed (full coverage)", x, y)
			}
			if !c.IsHead() && NominalPorts(c.R) == 0 {
				return fmt.Errorf("cell (%d,%d): unknown rune %q", x, y, c.R)
			}
		}
	}

	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			c := b.At(x, y)
			eff := EffectivePorts(b, x, y)
			bits := popcount(eff)
			if c.IsHead() {
				if bits != 1 {
					return fmt.Errorf("cell (%d,%d): head must have exactly one path link (got %d)", x, y, bits)
				}
				continue
			}
			if bits != 1 && bits != 2 {
				return fmt.Errorf("cell (%d,%d): wire must have degree 1 (tail) or 2 (internal), got %d", x, y, bits)
			}
		}
	}

	seen := make([]bool, b.W*b.H)
	var headsPerComp int
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			idx := y*b.W + x
			if seen[idx] {
				continue
			}
			headsPerComp = 0
			queue := []struct{ x, y int }{{x, y}}
			seen[idx] = true
			for qi := 0; qi < len(queue); qi++ {
				cx, cy := queue[qi].x, queue[qi].y
				if b.At(cx, cy).IsHead() {
					headsPerComp++
				}
				eff := EffectivePorts(b, cx, cy)
				try := func(nx, ny int, port uint8) {
					if eff&port == 0 {
						return
					}
					nidx := ny*b.W + nx
					if seen[nidx] {
						return
					}
					seen[nidx] = true
					queue = append(queue, struct{ x, y int }{nx, ny})
				}
				try(cx, cy-1, PortN)
				try(cx+1, cy, PortE)
				try(cx, cy+1, PortS)
				try(cx-1, cy, PortW)
			}
			if headsPerComp != 1 {
				return fmt.Errorf("component starting at (%d,%d): want exactly 1 head, got %d", x, y, headsPerComp)
			}
		}
	}

	return nil
}

// ValidatePartialBoard checks mutual links, path shape, and one head per connected component
// for boards that may contain empty cells (background). Empty cells are ignored.
// Full-coverage levels should use ValidateBoard instead.
func ValidatePartialBoard(b Board) error {
	hasAny := false
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			c := b.At(x, y)
			if c.IsEmpty() {
				continue
			}
			hasAny = true
			if !c.IsHead() && NominalPorts(c.R) == 0 {
				return fmt.Errorf("cell (%d,%d): unknown rune %q", x, y, c.R)
			}
		}
	}
	if !hasAny {
		return fmt.Errorf("partial board: no non-empty cells")
	}

	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			c := b.At(x, y)
			if c.IsEmpty() {
				continue
			}
			eff := EffectivePorts(b, x, y)
			bits := popcount(eff)
			if c.IsHead() {
				if bits != 1 {
					return fmt.Errorf("cell (%d,%d): head must have exactly one path link (got %d)", x, y, bits)
				}
				continue
			}
			if bits != 1 && bits != 2 {
				return fmt.Errorf("cell (%d,%d): wire must have degree 1 (tail) or 2 (internal), got %d", x, y, bits)
			}
		}
	}

	seen := make([]bool, b.W*b.H)
	var headsPerComp int
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			idx := y*b.W + x
			if seen[idx] || b.At(x, y).IsEmpty() {
				continue
			}
			headsPerComp = 0
			queue := []struct{ x, y int }{{x, y}}
			seen[idx] = true
			for qi := 0; qi < len(queue); qi++ {
				cx, cy := queue[qi].x, queue[qi].y
				if b.At(cx, cy).IsHead() {
					headsPerComp++
				}
				eff := EffectivePorts(b, cx, cy)
				try := func(nx, ny int, port uint8) {
					if eff&port == 0 {
						return
					}
					nidx := ny*b.W + nx
					if seen[nidx] {
						return
					}
					seen[nidx] = true
					queue = append(queue, struct{ x, y int }{nx, ny})
				}
				try(cx, cy-1, PortN)
				try(cx+1, cy, PortE)
				try(cx, cy+1, PortS)
				try(cx-1, cy, PortW)
			}
			if headsPerComp != 1 {
				return fmt.Errorf("component starting at (%d,%d): want exactly 1 head, got %d", x, y, headsPerComp)
			}
		}
	}

	return nil
}

func popcount(m uint8) int {
	n := 0
	for m != 0 {
		n++
		m &= m - 1
	}
	return n
}

// PathFromHead returns all cells on the polyline containing the head at (hx,hy), including the head.
func PathFromHead(b Board, hx, hy int) ([]struct{ X, Y int }, error) {
	if !b.InBounds(hx, hy) || !b.At(hx, hy).IsHead() {
		return nil, fmt.Errorf("not a head at (%d,%d)", hx, hy)
	}
	heff := EffectivePorts(b, hx, hy)
	if popcount(heff) != 1 {
		return nil, fmt.Errorf("head at (%d,%d): expected one body link", hx, hy)
	}
	bx, by, ok := stepAlongPorts(b, hx, hy, heff, -1, -1)
	if !ok {
		return nil, fmt.Errorf("head at (%d,%d): no body neighbor", hx, hy)
	}

	out := []struct{ X, Y int }{{hx, hy}}
	px, py := hx, hy
	cx, cy := bx, by
	for {
		out = append(out, struct{ X, Y int }{cx, cy})
		eff := EffectivePorts(b, cx, cy)
		var nx, ny int
		var found bool
		try := func(tx, ty int, needPort uint8) {
			if eff&needPort == 0 || found {
				return
			}
			if tx == px && ty == py {
				return
			}
			if !b.InBounds(tx, ty) {
				return
			}
			dir := directionFromTo(cx, cy, tx, ty)
			if !linked(b.At(cx, cy), b.At(tx, ty), dir) {
				return
			}
			nx, ny = tx, ty
			found = true
		}
		try(cx, cy-1, PortN)
		try(cx+1, cy, PortE)
		try(cx, cy+1, PortS)
		try(cx-1, cy, PortW)
		if !found {
			break
		}
		px, py, cx, cy = cx, cy, nx, ny
	}
	return out, nil
}

// stepAlongPorts picks a neighbor reachable from (x,y) along eff, excluding (px,py).
func stepAlongPorts(b Board, x, y int, eff uint8, px, py int) (int, int, bool) {
	cands := []struct {
		tx, ty int
		port   uint8
	}{
		{x, y - 1, PortN},
		{x + 1, y, PortE},
		{x, y + 1, PortS},
		{x - 1, y, PortW},
	}
	for _, c := range cands {
		if eff&c.port == 0 {
			continue
		}
		if c.tx == px && c.ty == py {
			continue
		}
		if !b.InBounds(c.tx, c.ty) {
			continue
		}
		dir := directionFromTo(x, y, c.tx, c.ty)
		if !linked(b.At(x, y), b.At(c.tx, c.ty), dir) {
			continue
		}
		return c.tx, c.ty, true
	}
	return 0, 0, false
}

func directionFromTo(x0, y0, x1, y1 int) Direction {
	switch {
	case x1 == x0 && y1 == y0-1:
		return North
	case x1 == x0+1 && y1 == y0:
		return East
	case x1 == x0 && y1 == y0+1:
		return South
	case x1 == x0-1 && y1 == y0:
		return West
	default:
		return North
	}
}
