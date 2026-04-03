package game

// VerifyGreedyFirstClearsBoard returns true iff repeated greedy clearing removes every arrow:
// in row-major order (y then x), repeatedly fire the first head whose ray escapes until the board is empty.
// Background empty cells are unchanged; Won is when no arrow cells remain.
func VerifyGreedyFirstClearsBoard(b Board) bool {
	g := NewGame(b.Clone(), 1<<20, "")
	for !g.Won() {
		found := false
		for y := 0; y < g.Board.H && !found; y++ {
			for x := 0; x < g.Board.W && !found; x++ {
				if !g.Board.At(x, y).IsHead() || !RayEscapes(g.Board, x, y) {
					continue
				}
				if TryFire(g, x, y) != FireCleared {
					return false
				}
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// VerifySolvable reports whether some sequence of legal fires clears the board.
// It uses backtracking (exponential); intended for tests and small boards.
func VerifySolvable(b Board) bool {
	g := NewGame(b, 1<<20, "")
	return verifySolvableRec(g)
}

// verifySolvableRec explores firing sequences via depth-first search (mutates cloned games).
func verifySolvableRec(g *Game) bool {
	if g.Won() {
		return true
	}
	type xy struct{ x, y int }
	var heads []xy
	for y := 0; y < g.Board.H; y++ {
		for x := 0; x < g.Board.W; x++ {
			if g.Board.At(x, y).IsHead() && RayEscapes(g.Board, x, y) {
				heads = append(heads, xy{x, y})
			}
		}
	}
	if len(heads) == 0 {
		return false
	}
	for _, h := range heads {
		gc := NewGame(g.Board, g.Lives, g.LevelName)
		TryFire(gc, h.x, h.y)
		if verifySolvableRec(gc) {
			return true
		}
	}
	return false
}
