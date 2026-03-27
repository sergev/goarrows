package game

// Game is playable state: board, lives, and identity for UI.
type Game struct {
	Board     Board
	Lives     int
	LevelName string
}

// NewGame returns a copy of the board as the initial state.
func NewGame(b Board, lives int, levelName string) *Game {
	bc := b.Clone()
	return &Game{
		Board:     bc,
		Lives:     lives,
		LevelName: levelName,
	}
}

// Reset replaces the board and lives from a solved template.
func (g *Game) Reset(template Board, lives int) {
	g.Board = template.Clone()
	g.Lives = lives
}
