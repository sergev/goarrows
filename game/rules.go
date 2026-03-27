package game

// FireResult describes the outcome of TryFire.
type FireResult int8

const (
	FireNone FireResult = iota // empty, non-head, or invalid
	FireCleared                // path escaped and was removed
	FireBlocked                // ray blocked, life lost
)

// RayEscapes reports whether the head at (x, y) can fire off the board.
func RayEscapes(b Board, x, y int) bool {
	c := b.At(x, y)
	if !c.IsHead() {
		return false
	}
	fire, ok := HeadFireDir(c.R)
	if !ok {
		return false
	}
	dx, dy := Delta(fire)
	for cx, cy := x+dx, y+dy; b.InBounds(cx, cy); cx, cy = cx+dx, cy+dy {
		if !b.At(cx, cy).IsEmpty() {
			return false
		}
	}
	return true
}

// TryFire attempts to fire the head at (x, y). Non-head or empty: FireNone.
// Head with clear ray: removes the entire polyline. Head blocked: FireBlocked.
func TryFire(g *Game, x, y int) FireResult {
	if !g.Board.InBounds(x, y) {
		return FireNone
	}
	c := g.Board.At(x, y)
	if c.IsEmpty() || !c.IsHead() {
		return FireNone
	}
	if !RayEscapes(g.Board, x, y) {
		if g.Lives > 0 {
			g.Lives--
		}
		return FireBlocked
	}
	path, err := PathFromHead(g.Board, x, y)
	if err != nil {
		return FireNone
	}
	for _, p := range path {
		g.Board.Set(p.X, p.Y, Cell{})
	}
	return FireCleared
}

func (g *Game) Won() bool {
	return g.Board.NonEmptyCount() == 0
}

func (g *Game) Lost() bool {
	return g.Lives <= 0 && g.Board.NonEmptyCount() > 0
}
