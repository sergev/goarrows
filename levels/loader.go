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

// Pack is an ordered set of named levels from the embedded data.
type Pack struct {
	Names  []string
	Boards []game.Board
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
