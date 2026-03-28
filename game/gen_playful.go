package game

// acceptPlayful may filter boards for procedural quality. Inverse polyline placement is already
// structurally varied; keep this permissive so generation terminates quickly.
func acceptPlayful(b Board, order []placedComponent, w, h int) bool {
	_ = b
	_ = order
	_ = w
	_ = h
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
