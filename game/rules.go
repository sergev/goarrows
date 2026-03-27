package game

// FireResult describes the outcome of TryFire.
type FireResult int8

const (
	FireNone FireResult = iota // empty cell or no change to board
	FireCleared                // arrow escaped and was removed
	FireBlocked                // path blocked, life lost
)

// RayEscapes reports whether an arrow at (x, y) can exit: every cell along
// the ray until the board edge must be empty (the starting cell is the arrow).
func RayEscapes(b Board, x, y int) bool {
	c := b.At(x, y)
	if c.Empty {
		return false
	}
	dx, dy := Delta(c.Dir)
	for cx, cy := x+dx, y+dy; b.InBounds(cx, cy); cx, cy = cx+dx, cy+dy {
		if !b.At(cx, cy).Empty {
			return false
		}
	}
	return true
}

// TryFire attempts to fire the arrow at (x, y). Empty cell: FireNone, no life change.
// Arrow with clear path: removes arrow, FireCleared. Arrow blocked: FireBlocked, lives decremented.
func TryFire(g *Game, x, y int) FireResult {
	if !g.Board.InBounds(x, y) {
		return FireNone
	}
	c := g.Board.At(x, y)
	if c.Empty {
		return FireNone
	}
	if RayEscapes(g.Board, x, y) {
		g.Board.Set(x, y, Cell{Empty: true})
		return FireCleared
	}
	if g.Lives > 0 {
		g.Lives--
	}
	return FireBlocked
}

func (g *Game) Won() bool {
	return g.Board.ArrowCount() == 0
}

func (g *Game) Lost() bool {
	return g.Lives <= 0 && g.Board.ArrowCount() > 0
}
