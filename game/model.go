package game

// Direction is the facing of a head’s fire ray.
type Direction int8

const (
	North Direction = iota
	East
	South
	West
)

// Cell holds a display rune: 0 = empty, otherwise wire (─│┌┐└┘) or head (^v<> / ▲▼◀▶).
type Cell struct {
	R rune
}

// IsEmpty reports whether the cell has no glyph (R == 0).
func (c Cell) IsEmpty() bool {
	return c.R == 0
}

// Board is a rectangular grid of cells, row-major (y then x).
type Board struct {
	W, H int
	Data []Cell
}

// NewBoard allocates an empty w×h board (all cells empty).
func NewBoard(w, h int) Board {
	return Board{W: w, H: h, Data: make([]Cell, w*h)}
}

// InBounds reports whether (x, y) is a valid index in row-major order.
func (b Board) InBounds(x, y int) bool {
	return x >= 0 && x < b.W && y >= 0 && y < b.H
}

// At returns the cell at (x, y); callers should use InBounds first for safety.
func (b Board) At(x, y int) Cell {
	return b.Data[y*b.W+x]
}

// Set writes cell c at (x, y).
func (b *Board) Set(x, y int, c Cell) {
	b.Data[y*b.W+x] = c
}

// Clone returns a deep copy of the board grid.
func (b Board) Clone() Board {
	cp := NewBoard(b.W, b.H)
	copy(cp.Data, b.Data)
	return cp
}

// NonEmptyCount counts cells still occupied by path material.
func (b Board) NonEmptyCount() int {
	n := 0
	for _, c := range b.Data {
		if !c.IsEmpty() {
			n++
		}
	}
	return n
}

// Delta maps a cardinal direction to grid deltas (x increases east, y increases south).
func Delta(d Direction) (dx, dy int) {
	switch d {
	case North:
		return 0, -1
	case East:
		return 1, 0
	case South:
		return 0, 1
	case West:
		return -1, 0
	default:
		return 0, 0
	}
}
