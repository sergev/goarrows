package game

// VerifySolvable reports whether some sequence of legal fires clears the board.
// It uses backtracking (exponential); intended for tests and small boards.
func VerifySolvable(b Board) bool {
	g := NewGame(b, 1<<20, "")
	return verifySolvableRec(g)
}

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
