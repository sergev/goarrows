package game

// Direction is the facing of an arrow on the board.
type Direction int8

const (
	North Direction = iota
	East
	South
	West
)

// Cell is either empty or an arrow facing a direction.
type Cell struct {
	Empty bool
	Dir   Direction
}

// Board is a rectangular grid of cells, row-major (y then x).
type Board struct {
	W, H int
	Data []Cell
}

func NewBoard(w, h int) Board {
	return Board{W: w, H: h, Data: make([]Cell, w*h)}
}

func (b Board) InBounds(x, y int) bool {
	return x >= 0 && x < b.W && y >= 0 && y < b.H
}

func (b Board) At(x, y int) Cell {
	return b.Data[y*b.W+x]
}

func (b *Board) Set(x, y int, c Cell) {
	b.Data[y*b.W+x] = c
}

func (b Board) Clone() Board {
	cp := NewBoard(b.W, b.H)
	copy(cp.Data, b.Data)
	return cp
}

func (b Board) ArrowCount() int {
	n := 0
	for _, c := range b.Data {
		if !c.Empty {
			n++
		}
	}
	return n
}

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
