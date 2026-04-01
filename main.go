package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"goarrows/game"
	"goarrows/levels"
	"goarrows/ui"
)

// fireOverlay is a centered modal after a fire outcome (win or game over).
type fireOverlay struct {
	positive bool
	lines    []string
}

// fireUIResult combines an optional status line (e.g. Blocked) with an optional modal.
type fireUIResult struct {
	status  string
	overlay *fireOverlay
}

type animState struct {
	active   bool
	hidePath []struct{ X, Y int } // original fired path (masked during animation)
	frames   []ui.FireAnimOverlay // precomputed snake frames
	step     int
	nextStep time.Time
	fireX    int
	fireY    int
}

// optionalInt64Flag is a flag.Value for -seed: unset means "not provided on CLI".
type optionalInt64Flag struct {
	set   bool
	value int64
}

func (o *optionalInt64Flag) String() string {
	if !o.set {
		return ""
	}
	return strconv.FormatInt(o.value, 10)
}

func (o *optionalInt64Flag) Set(s string) error {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	o.value = v
	o.set = true
	return nil
}

func resolveProceduralSeed(f *optionalInt64Flag) int64 {
	if f.set {
		return f.value
	}
	if testing.Testing() {
		return 0
	}
	return time.Now().UnixNano()
}

func main() {
	startLives := flag.Int("lives", 3, "starting lives per level (use -1 for unlimited)")
	levelPath := flag.String("level", "", "path to a single level file (optional; default: procedural pack)")
	seedFlag := &optionalInt64Flag{}
	flag.Var(seedFlag, "seed", "base RNG seed for procedural levels (omit for random from clock; -seed 0 fixes zero)")
	gen := flag.String("gen", game.GenGrow, "procedural generation algorithm: grow (default), inverse")
	flag.Parse()

	if err := game.ValidateGenAlgorithm(*gen); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	seed := resolveProceduralSeed(seedFlag)
	pack, err := loadPack(*levelPath, seed, *gen)
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
	helpSt := def.Foreground(tcell.ColorGray)

	showHelp := false
	status := ""
	var modal *fireOverlay
	generatingN := 0
	var anim animState
	animStep := 75 * time.Millisecond

	idx := 0
	var g *game.Game
	cx, cy := 0, 0

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

		var fireAnim *ui.FireAnimOverlay
		if anim.active && anim.step < len(anim.frames) {
			fireAnim = &anim.frames[anim.step]
		}
		ui.DrawGrid(s, ox, oy, g.Board, cx, cy, base, cursorSt, fireAnim)

		lineY := oy + gh + 1
		if lineY >= sh {
			lineY = sh - 1
		}
		drawStr(s, 0, lineY, sw, fmt.Sprintf(" %s", g.LevelName), titleSt)
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
			} else if strings.HasPrefix(status, "Cleared") {
				st = winSt
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
		if modal != nil {
			drawFireOverlay(s, sw, sh, modal, def)
		}
		if generatingN > 0 {
			drawGeneratingOverlay(s, sw, sh, generatingN, def)
		}
		s.Show()
	}

	g = newGameWithGenOverlay(pack, idx, *startLives, &generatingN, redraw)
	clampCursor(g, &cx, &cy)
	redraw()
	ticker := time.NewTicker(16 * time.Millisecond)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			s.PostEvent(tcell.NewEventInterrupt(nil))
		}
	}()

	quit := false
	for !quit {
		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			s.Sync()
			redraw()
		case *tcell.EventInterrupt:
			if !anim.active || time.Now().Before(anim.nextStep) {
				continue
			}
			anim.step++
			anim.nextStep = time.Now().Add(animStep)
			if anim.step >= len(anim.frames) {
				anim.active = false
				fr := applyFire(g, anim.fireX, anim.fireY, *startLives)
				status, modal = fr.status, fr.overlay
			}
			redraw()
		case *tcell.EventKey:
			if showHelp {
				showHelp = false
				redraw()
				continue
			}
			if anim.active {
				switch ev.Key() {
				case tcell.KeyCtrlC, tcell.KeyEscape:
					quit = true
				case tcell.KeyRune:
					if r := ev.Rune(); r == 'q' || r == 'Q' {
						quit = true
					}
				}
				if quit {
					continue
				}
				redraw()
				continue
			}
			if g.Won() || g.Lost() {
				switch ev.Key() {
				case tcell.KeyCtrlC, tcell.KeyEscape:
					quit = true
				case tcell.KeyEnter:
					if g.Won() {
						idx = (idx + 1) % pack.Len()
						g = newGameWithGenOverlay(pack, idx, *startLives, &generatingN, redraw)
						status, modal = "", nil
						anim.active = false
						clampCursor(g, &cx, &cy)
					}
				case tcell.KeyRune:
					switch ev.Rune() {
					case 'q', 'Q':
						quit = true
					case 'r', 'R':
						resetLevel(pack, &g, idx, *startLives)
						status, modal = "", nil
						anim.active = false
						clampCursor(g, &cx, &cy)
					case 'n', 'N':
						idx = (idx + 1) % pack.Len()
						g = newGameWithGenOverlay(pack, idx, *startLives, &generatingN, redraw)
						status, modal = "", nil
						anim.active = false
						clampCursor(g, &cx, &cy)
					case 'p', 'P':
						idx = (idx - 1 + pack.Len()) % pack.Len()
						g = newGameWithGenOverlay(pack, idx, *startLives, &generatingN, redraw)
						status, modal = "", nil
						anim.active = false
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
				started := tryStartFireAnimation(g, cx, cy, &anim, animStep)
				if !started {
					fr := applyFire(g, cx, cy, *startLives)
					status, modal = fr.status, fr.overlay
				}
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
					started := tryStartFireAnimation(g, cx, cy, &anim, animStep)
					if !started {
						fr := applyFire(g, cx, cy, *startLives)
						status, modal = fr.status, fr.overlay
					}
				case 'r', 'R':
					resetLevel(pack, &g, idx, *startLives)
					status, modal = "", nil
					anim.active = false
				case 'n', 'N':
					idx = (idx + 1) % pack.Len()
					g = newGameWithGenOverlay(pack, idx, *startLives, &generatingN, redraw)
					status, modal = "", nil
					anim.active = false
					clampCursor(g, &cx, &cy)
				case 'p', 'P':
					idx = (idx - 1 + pack.Len()) % pack.Len()
					g = newGameWithGenOverlay(pack, idx, *startLives, &generatingN, redraw)
					status, modal = "", nil
					anim.active = false
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

func loadPack(singlePath string, seed int64, gen string) (*levels.Pack, error) {
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
	p := levels.NewProceduralPack(seed, gen)
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

func newGameWithGenOverlay(pack *levels.Pack, idx, startLives int, generatingN *int, redraw func()) *game.Game {
	n := pack.ProceduralSideLen(idx)
	if n > 0 && !pack.ProceduralLevelReady(idx) {
		*generatingN = n
		redraw()
	}
	g := newGameForLevel(pack, idx, startLives)
	*generatingN = 0
	return g
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

func applyFire(g *game.Game, cx, cy, startLives int) fireUIResult {
	if g.Won() || g.Lost() {
		return fireUIResult{}
	}
	switch game.TryFire(g, cx, cy) {
	case game.FireNone:
		return fireUIResult{}
	case game.FireCleared:
		if g.Won() {
			return fireUIResult{
				overlay: &fireOverlay{
					positive: true,
					lines: []string{
						"You win!",
						"",
						"Press Enter for next level",
						"",
						"n next  p prev  r replay  q quit",
					},
				},
			}
		}
		return fireUIResult{status: "Cleared."}
	case game.FireBlocked:
		if g.Lost() {
			return fireUIResult{
				overlay: &fireOverlay{
					positive: false,
					lines: []string{
						"Game over",
						"",
						"r restart  q quit",
					},
				},
			}
		}
		return fireUIResult{status: "Blocked!"}
	default:
		return fireUIResult{}
	}
}

func tryStartFireAnimation(g *game.Game, cx, cy int, anim *animState, stepDur time.Duration) bool {
	if g.Won() || g.Lost() || !g.Board.InBounds(cx, cy) {
		return false
	}
	c := g.Board.At(cx, cy)
	if c.IsEmpty() || !c.IsHead() || !game.RayEscapes(g.Board, cx, cy) {
		return false
	}
	path, err := game.PathFromHead(g.Board, cx, cy)
	if err != nil || len(path) == 0 {
		return false
	}
	cells := fireTravelCells(g.Board, cx, cy)
	if len(cells) == 0 {
		return false
	}
	frames, ok := buildPointerFrames(g.Board, path, cells, c.R)
	if !ok || len(frames) == 0 {
		return false
	}
	anim.active = true
	anim.hidePath = path
	anim.frames = frames
	anim.step = 0
	anim.nextStep = time.Now().Add(stepDur)
	anim.fireX = cx
	anim.fireY = cy
	return true
}

func buildPointerFrames(b game.Board, path, ray []struct{ X, Y int }, headRune rune) ([]ui.FireAnimOverlay, bool) {
	if len(path) == 0 || len(ray) == 0 {
		return nil, false
	}
	fireDir, ok := game.HeadFireDir(headRune)
	if !ok {
		return nil, false
	}
	dx, dy := game.Delta(fireDir)
	bodyRune := straightBodyRune(fireDir)
	cur := make([]ui.OverlayCell, len(path))
	for i, p := range path {
		cur[i] = ui.OverlayCell{X: p.X, Y: p.Y, R: b.At(p.X, p.Y).R}
	}
	// Keep animating after head reaches the boundary cell so the tail also
	// reaches and exits the boundary before we commit final clear.
	totalSteps := len(ray) + len(path) - 1
	frames := make([]ui.FireAnimOverlay, 0, totalSteps)
	for step := 1; step <= totalSteps; step++ {
		if len(cur) == 0 {
			break
		}
		cur[0].R = bodyRune
		hx, hy := headPositionForStep(ray, dx, dy, step)
		next := ui.OverlayCell{X: hx, Y: hy, R: headRune}
		nxt := make([]ui.OverlayCell, 0, len(cur))
		nxt = append(nxt, next)
		if len(cur) > 1 {
			nxt = append(nxt, cur[:len(cur)-1]...)
		}
		cur = nxt
		frameCells := make([]ui.OverlayCell, len(cur))
		copy(frameCells, cur)
		frames = append(frames, ui.FireAnimOverlay{
			HidePath: path,
			Cells:    frameCells,
		})
	}
	return frames, len(frames) > 0
}

func headPositionForStep(ray []struct{ X, Y int }, dx, dy, step int) (int, int) {
	if step <= len(ray) {
		p := ray[step-1]
		return p.X, p.Y
	}
	last := ray[len(ray)-1]
	extra := step - len(ray)
	return last.X + extra*dx, last.Y + extra*dy
}

func straightBodyRune(d game.Direction) rune {
	switch d {
	case game.North, game.South:
		return '│'
	default:
		return '─'
	}
}

func fireTravelCells(b game.Board, cx, cy int) []struct{ X, Y int } {
	c := b.At(cx, cy)
	fire, ok := game.HeadFireDir(c.R)
	if !ok {
		return nil
	}
	dx, dy := game.Delta(fire)
	var out []struct{ X, Y int }
	for x, y := cx+dx, cy+dy; b.InBounds(x, y); x, y = x+dx, y+dy {
		out = append(out, struct{ X, Y int }{X: x, Y: y})
	}
	return out
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

func drawGeneratingOverlay(s tcell.Screen, sw, sh, n int, fill tcell.Style) {
	if n <= 0 {
		return
	}
	lines := []string{fmt.Sprintf(" Creating Level %d×%d... ", n, n)}
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

func drawFireOverlay(s tcell.Screen, sw, sh int, o *fireOverlay, fill tcell.Style) {
	if o == nil || len(o.lines) == 0 {
		return
	}
	bg := tcell.ColorDarkOliveGreen
	if !o.positive {
		bg = tcell.ColorDarkRed
	}
	st := fill.Foreground(tcell.ColorWhite).Background(bg)
	lines := o.lines
	boxW := 0
	for _, ln := range lines {
		if w := len([]rune(ln)); w > boxW {
			boxW = w
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
