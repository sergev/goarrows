package game

import (
	"math"
)

// acceptPlayful filters structurally valid boards using cheap heuristics so levels are
// not trivial (too many immediate clears) nor overly repetitive (length/turn variety).
// Single-component boards and small grids skip strict checks so generation always terminates.
func acceptPlayful(b Board, order []placedComponent, w, h int) bool {
	wh := w * h
	if len(order) <= 1 {
		return true
	}
	if wh <= 16 {
		return true
	}
	// Avoid the old dominant pattern: exactly two long snakes on medium/large boards.
	if wh >= 24 && len(order) == 2 {
		return false
	}

	heads := countHeadsOnBoard(b)
	if heads < 2 {
		return true
	}

	esc := countInitialRayEscapes(b)
	// Too many heads that can fire immediately → boring / too easy (only pressure large head counts).
	if heads >= 10 {
		maxEsc := minInt(heads-1, maxInt(6, (heads*4)/5+int(math.Ceil(math.Sqrt(float64(wh))))))
		if esc > maxEsc {
			return false
		}
	}
	// Very large boards: require at least one obvious escape so the level isn't hopelessly blocked.
	minEsc := 0
	if wh >= 64 {
		minEsc = 1
	}
	if esc < minEsc {
		return false
	}

	if !lengthDiversityOK(order, wh) {
		return false
	}
	if !turnDiversityOK(order, wh) {
		return false
	}
	return true
}

func countHeadsOnBoard(b Board) int {
	n := 0
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			if b.At(x, y).IsHead() {
				n++
			}
		}
	}
	return n
}

func countInitialRayEscapes(b Board) int {
	n := 0
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			if b.At(x, y).IsHead() && RayEscapes(b, x, y) {
				n++
			}
		}
	}
	return n
}

func lengthDiversityOK(order []placedComponent, wh int) bool {
	if len(order) < 3 {
		return true
	}
	minL, maxL := len(order[0].path), len(order[0].path)
	for i := 1; i < len(order); i++ {
		L := len(order[i].path)
		if L < minL {
			minL = L
		}
		if L > maxL {
			maxL = L
		}
	}
	if wh >= 48 && len(order) >= 4 && maxL <= 4 && maxL-minL <= 1 {
		return false
	}
	return true
}

func turnDiversityOK(order []placedComponent, wh int) bool {
	if wh < 48 {
		return true
	}
	t := totalPathTurns(order)
	if t < 4 {
		return false
	}
	return true
}

func totalPathTurns(order []placedComponent) int {
	t := 0
	for i := range order {
		t += pathTurnCount(order[i].path)
	}
	return t
}

// pathTurnCount counts vertices where the polyline changes direction (straight segments score 0).
func pathTurnCount(path []point) int {
	if len(path) < 3 {
		return 0
	}
	n := 0
	for i := 1; i < len(path)-1; i++ {
		dIn := dirFromTo(path[i-1].x, path[i-1].y, path[i].x, path[i].y)
		dOut := dirFromTo(path[i].x, path[i].y, path[i+1].x, path[i+1].y)
		if dIn != dOut {
			n++
		}
	}
	return n
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
