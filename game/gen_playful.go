package game

// countInitialRayEscapes counts heads whose firing ray reaches the board edge with no obstruction.
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
