package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"goarrows/game"
	"goarrows/levels"
	"goarrows/ui"
)

func main() {
	startLives := flag.Int("lives", 3, "starting lives per level (use -1 for unlimited)")
	levelPath := flag.String("level", "", "path to a single level file (optional; default: procedural pack)")
	seed := flag.Int64("seed", 0, "RNG seed for procedural levels (default: 0)")
	flag.Parse()

	pack, err := loadPack(*levelPath, *seed)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *startLives < -1 || *startLives == 0 {
		fmt.Fprintln(os.Stderr, "lives must be >= 1 or -1 for unlimited")
		os.Exit(1)
	}

	s, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := s.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer s.Fini()
	s.HideCursor()

	def := tcell.StyleDefault
	base := def.Foreground(tcell.ColorSilver).Background(tcell.ColorBlack)
	cursorSt := def.Foreground(tcell.ColorBlack).Background(tcell.ColorAqua).Bold(true)
	titleSt := def.Foreground(tcell.ColorYellow).Bold(true)
	msgSt := def.Foreground(tcell.ColorWhite)
	blockedSt := def.Foreground(tcell.ColorOrange).Bold(true)
	winSt := def.Foreground(tcell.ColorGreen).Bold(true)
	loseSt := def.Foreground(tcell.ColorRed).Bold(true)
	helpSt := def.Foreground(tcell.ColorGray)

	idx := 0
	g := newGameForLevel(pack, idx, *startLives)
	cx, cy := 0, 0
	clampCursor(g, &cx, &cy)
	showHelp := false
	status := ""

	redraw := func() {
		s.Clear()
		sw, sh := s.Size()
		gw, gh := ui.GridSize(g.Board.W, g.Board.H)
		hudLines := 4
		oy := (sh - gh - hudLines) / 2
		if oy < 0 {
			oy = 0
		}
		ox := (sw - gw) / 2
		if ox < 0 {
			ox = 0
		}

		ui.DrawGrid(s, ox, oy, g.Board, cx, cy, base, cursorSt)

		lineY := oy + gh + 1
		if lineY >= sh {
			lineY = sh - 1
		}
		drawStr(s, 0, lineY, sw, fmt.Sprintf(" %s  [%d/%d]", g.LevelName, idx+1, pack.Len()), titleSt)
		lineY++
		if lineY < sh {
			livesStr := formatLives(g.Lives, *startLives)
			left := fmt.Sprintf(" Lives: %s   Cells: %d", livesStr, g.Board.NonEmptyCount())
			drawStr(s, 0, lineY, sw, left, msgSt)
		}
		lineY++
		if lineY < sh && status != "" {
			st := msgSt
			if strings.HasPrefix(status, "Blocked") {
				st = blockedSt
			} else if strings.HasPrefix(status, "You win") || strings.HasPrefix(status, "Cleared") {
				st = winSt
			} else if strings.HasPrefix(status, "Game over") {
				st = loseSt
			}
			drawStr(s, 0, lineY, sw, " "+status, st)
		}
		lineY++
		if lineY < sh {
			drawStr(s, 0, lineY, sw, " hjkl/←↑↓→ move  space/enter fire  r restart  n/p level  ? help  q quit", helpSt)
		}

		if showHelp {
			drawHelpOverlay(s, sw, sh, base)
		}
		s.Show()
	}

	redraw()

	quit := false
	for !quit {
		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
			redraw()
		case *tcell.EventKey:
			if showHelp {
				showHelp = false
				redraw()
				continue
			}
			if g.Won() || g.Lost() {
				switch ev.Key() {
				case tcell.KeyCtrlC, tcell.KeyEscape:
					quit = true
				case tcell.KeyRune:
					switch ev.Rune() {
					case 'q', 'Q':
						quit = true
					case 'r', 'R':
						resetLevel(pack, &g, idx, *startLives)
						status = ""
						clampCursor(g, &cx, &cy)
					case 'n', 'N':
						idx = (idx + 1) % pack.Len()
						g = newGameForLevel(pack, idx, *startLives)
						status = ""
						clampCursor(g, &cx, &cy)
					case 'p', 'P':
						idx = (idx - 1 + pack.Len()) % pack.Len()
						g = newGameForLevel(pack, idx, *startLives)
						status = ""
						clampCursor(g, &cx, &cy)
					}
				}
				redraw()
				continue
			}

			switch ev.Key() {
			case tcell.KeyCtrlC:
				quit = true
			case tcell.KeyUp:
				moveCursor(g, &cx, &cy, 0, -1)
			case tcell.KeyDown:
				moveCursor(g, &cx, &cy, 0, 1)
			case tcell.KeyLeft:
				moveCursor(g, &cx, &cy, -1, 0)
			case tcell.KeyRight:
				moveCursor(g, &cx, &cy, 1, 0)
			case tcell.KeyEnter:
				status = applyFire(g, cx, cy, *startLives)
			case tcell.KeyRune:
				switch r := ev.Rune(); r {
				case 'q', 'Q':
					quit = true
				case 'h':
					moveCursor(g, &cx, &cy, -1, 0)
				case 'l':
					moveCursor(g, &cx, &cy, 1, 0)
				case 'k':
					moveCursor(g, &cx, &cy, 0, -1)
				case 'j':
					moveCursor(g, &cx, &cy, 0, 1)
				case ' ', 'f', 'F':
					status = applyFire(g, cx, cy, *startLives)
				case 'r', 'R':
					resetLevel(pack, &g, idx, *startLives)
					status = ""
				case 'n', 'N':
					idx = (idx + 1) % pack.Len()
					g = newGameForLevel(pack, idx, *startLives)
					status = ""
					clampCursor(g, &cx, &cy)
				case 'p', 'P':
					idx = (idx - 1 + pack.Len()) % pack.Len()
					g = newGameForLevel(pack, idx, *startLives)
					status = ""
					clampCursor(g, &cx, &cy)
				case '?':
					showHelp = true
				}
			}
			clampCursor(g, &cx, &cy)
			redraw()
		}
	}
}

func loadPack(singlePath string, seed int64) (*levels.Pack, error) {
	if singlePath != "" {
		b, err := levels.LoadFile(singlePath)
		if err != nil {
			return nil, err
		}
		name := singlePath
		if i := strings.LastIndexAny(singlePath, `/\`); i >= 0 {
			name = singlePath[i+1:]
		}
		return &levels.Pack{
			Names:  []string{name},
			Boards: []game.Board{b},
		}, nil
	}
	p := levels.NewProceduralPack(seed)
	if _, _, err := p.LevelAt(0); err != nil {
		return nil, err
	}
	return p, nil
}

func newGameForLevel(p *levels.Pack, idx, startLives int) *game.Game {
	b, name, err := p.LevelAt(idx)
	if err != nil {
		panic(err)
	}
	lives := startLives
	if lives < 0 {
		lives = 1 << 30
	}
	return game.NewGame(b, lives, name)
}

func resetLevel(p *levels.Pack, g **game.Game, idx, startLives int) {
	b, _, err := p.LevelAt(idx)
	if err != nil {
		panic(err)
	}
	lives := startLives
	if lives < 0 {
		lives = 1 << 30
	}
	(*g).Reset(b, lives)
}

func clampCursor(g *game.Game, cx, cy *int) {
	if *cx >= g.Board.W {
		*cx = g.Board.W - 1
	}
	if *cy >= g.Board.H {
		*cy = g.Board.H - 1
	}
	if *cx < 0 {
		*cx = 0
	}
	if *cy < 0 {
		*cy = 0
	}
}

func moveCursor(g *game.Game, cx, cy *int, dx, dy int) {
	*cx += dx
	*cy += dy
	clampCursor(g, cx, cy)
}

func applyFire(g *game.Game, cx, cy, startLives int) string {
	if g.Won() || g.Lost() {
		return ""
	}
	switch game.TryFire(g, cx, cy) {
	case game.FireNone:
		return ""
	case game.FireCleared:
		if g.Won() {
			return "You win!  n next  p prev  r replay  q quit"
		}
		return "Cleared."
	case game.FireBlocked:
		if g.Lost() {
			return "Game over.  r restart  q quit"
		}
		return "Blocked!"
	default:
		return ""
	}
}

func formatLives(n, start int) string {
	if start < 0 {
		return "∞"
	}
	return fmt.Sprintf("%d", n)
}

func drawStr(s tcell.Screen, x0, y, maxW int, text string, st tcell.Style) {
	col := x0
	for _, r := range text {
		if col >= maxW {
			break
		}
		s.SetContent(col, y, r, nil, st)
		col++
	}
}

func drawHelpOverlay(s tcell.Screen, sw, sh int, fill tcell.Style) {
	lines := []string{
		" Arrows — TUI puzzle",
		"",
		" Fire an arrow (space/enter) to slide it off the board",
		" along its direction if the path is empty. If another",
		" arrow blocks the path, you lose a life.",
		"",
		" Win by clearing all arrows. Lose if lives hit 0.",
		"",
		" Any key closes this help.",
	}
	boxW := 0
	for _, ln := range lines {
		if len([]rune(ln)) > boxW {
			boxW = len([]rune(ln))
		}
	}
	boxW += 4
	boxH := len(lines) + 4
	ox := (sw - boxW) / 2
	oy := (sh - boxH) / 2
	if ox < 0 {
		ox = 0
	}
	if oy < 0 {
		oy = 0
	}
	st := fill.Foreground(tcell.ColorWhite).Background(tcell.ColorNavy)
	for j := 0; j < boxH; j++ {
		for i := 0; i < boxW && i < sw; i++ {
			x, y := ox+i, oy+j
			if x < 0 || y < 0 || y >= sh {
				continue
			}
			var r rune = ' '
			if j == 0 && i == 0 {
				r = '┌'
			} else if j == 0 && i == boxW-1 {
				r = '┐'
			} else if j == boxH-1 && i == 0 {
				r = '└'
			} else if j == boxH-1 && i == boxW-1 {
				r = '┘'
			} else if j == 0 || j == boxH-1 {
				r = '─'
			} else if i == 0 || i == boxW-1 {
				r = '│'
			}
			s.SetContent(x, y, r, nil, st)
		}
	}
	for li, ln := range lines {
		drawStr(s, ox+2, oy+2+li, ox+boxW-1, ln, st)
	}
}
