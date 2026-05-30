package game

// VerifySolvableFast reports whether some firing order clears every arrow.
// It is equivalent in acceptance to VerifyGreedyFirstClearsBoard (firing can only
// remove cells, so once a ray is clear it stays clear — any-order greedy and
// row-major greedy succeed on exactly the same set of boards) but runs in
// O(K + total_ray_length) instead of O(K²·N) by using a work-list and a per-cell
// reverse index from cells to the heads whose rays pass through them.
//
// heads must list every head cell on b exactly once. The constructive generator
// in gen_reverse.go already tracks heads, but every board it returns is solvable
// by construction so this function is reserved for tests and standalone callers.
func VerifySolvableFast(b Board, heads []Point) bool {
	K := len(heads)
	if K == 0 {
		return b.NonEmptyCount() == 0
	}

	rayCells := make([][]int, K)
	cellHeads := make([][]int, b.W*b.H)
	for k, hp := range heads {
		c := b.At(hp.X, hp.Y)
		if !c.IsHead() {
			return false
		}
		fire, ok := HeadFireDir(c.R)
		if !ok {
			return false
		}
		dx, dy := Delta(fire)
		for cx, cy := hp.X+dx, hp.Y+dy; b.InBounds(cx, cy); cx, cy = cx+dx, cy+dy {
			idx := cy*b.W + cx
			rayCells[k] = append(rayCells[k], idx)
			cellHeads[idx] = append(cellHeads[idx], k)
		}
	}

	scratch := make([]Cell, len(b.Data))
	copy(scratch, b.Data)

	rayClear := make([]bool, K)
	alive := make([]bool, K)
	for k := 0; k < K; k++ {
		alive[k] = true
	}
	queue := make([]int, 0, K)
	for k := 0; k < K; k++ {
		clearRay := true
		for _, idx := range rayCells[k] {
			if scratch[idx].R != 0 {
				clearRay = false
				break
			}
		}
		if clearRay {
			rayClear[k] = true
			queue = append(queue, k)
		}
	}

	remaining := K
	for len(queue) > 0 {
		k := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if !alive[k] {
			continue
		}
		hp := heads[k]
		path, err := pathFromHeadOn(scratch, b.W, b.H, hp.X, hp.Y)
		if err != nil {
			return false
		}
		for _, idx := range path {
			scratch[idx] = Cell{}
		}
		alive[k] = false
		remaining--
		for _, idx := range path {
			for _, k2 := range cellHeads[idx] {
				if rayClear[k2] || !alive[k2] {
					continue
				}
				clearRay := true
				for _, ridx := range rayCells[k2] {
					if scratch[ridx].R != 0 {
						clearRay = false
						break
					}
				}
				if clearRay {
					rayClear[k2] = true
					queue = append(queue, k2)
				}
			}
		}
	}
	return remaining == 0
}

// pathFromHeadOn walks the polyline starting at the head on a flat cell slice.
// It mirrors PathFromHead but avoids constructing a Board wrapper so the verifier
// can reuse a scratch buffer. Returns the flat cell indices of every path cell
// (head included).
func pathFromHeadOn(cells []Cell, w, h, hx, hy int) ([]int, error) {
	tmp := Board{W: w, H: h, Data: cells}
	pts, err := PathFromHead(tmp, hx, hy)
	if err != nil {
		return nil, err
	}
	out := make([]int, len(pts))
	for i, p := range pts {
		out[i] = p.Y*w + p.X
	}
	return out, nil
}

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
	var heads []Point
	for y := 0; y < g.Board.H; y++ {
		for x := 0; x < g.Board.W; x++ {
			if g.Board.At(x, y).IsHead() && RayEscapes(g.Board, x, y) {
				heads = append(heads, Point{x, y})
			}
		}
	}
	if len(heads) == 0 {
		return false
	}
	for _, h := range heads {
		gc := NewGame(g.Board, g.Lives, g.LevelName)
		TryFire(gc, h.X, h.Y)
		if verifySolvableRec(gc) {
			return true
		}
	}
	return false
}
