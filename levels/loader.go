package levels

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"

	"goarrows/game"
)

//go:embed data/*.txt
var data embed.FS

// Pack is an ordered set of levels from embedded files, a single file, or
// procedural generation (when proc is non-nil).
type Pack struct {
	Names   []string
	Boards  []game.Board
	proc    *proceduralSource
}

// NewProceduralPack returns a pack with unbounded levels: size (i+3)×(i+3)
// for index i, deterministic per seed. algorithm is passed to game.GenerateBoard
// (e.g. game.GenGrow or game.GenInverse).
func NewProceduralPack(seed int64, algorithm string) *Pack {
	return &Pack{proc: newProceduralSource(seed, algorithm)}
}

// Len returns the number of levels (large constant for procedural packs).
func (p *Pack) Len() int {
	if p.proc != nil {
		return ProceduralLevelCount
	}
	return len(p.Boards)
}

// LevelAt returns the board and display name for index i.
func (p *Pack) LevelAt(i int) (game.Board, string, error) {
	if p.proc != nil {
		return p.proc.levelAt(i)
	}
	if i < 0 || i >= len(p.Boards) {
		return game.Board{}, "", fmt.Errorf("level index %d out of range [0,%d)", i, len(p.Boards))
	}
	return p.Boards[i], p.Names[i], nil
}

// LoadEmbedded parses all embedded .txt levels (sorted by filename).
func LoadEmbedded() (*Pack, error) {
	entries, err := fs.ReadDir(data, "data")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	p := &Pack{
		Names:  make([]string, 0, len(names)),
		Boards: make([]game.Board, 0, len(names)),
	}
	for _, fn := range names {
		b, err := loadEmbeddedFile(path.Join("data", fn))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", fn, err)
		}
		p.Names = append(p.Names, strings.TrimSuffix(fn, ".txt"))
		p.Boards = append(p.Boards, b)
	}
	if len(p.Boards) == 0 {
		return nil, fmt.Errorf("no embedded levels")
	}
	return p, nil
}

func loadEmbeddedFile(name string) (game.Board, error) {
	raw, err := data.ReadFile(name)
	if err != nil {
		return game.Board{}, err
	}
	return game.ParseLevelString(string(raw))
}

// LoadFile loads a single level from a path (newline-separated grid).
func LoadFile(filePath string) (game.Board, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return game.Board{}, err
	}
	return game.ParseLevelString(string(raw))
}
