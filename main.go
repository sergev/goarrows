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

// fireUIResult combines an optional status line (e.g. Blocked) with an optional modal.
type fireUIResult struct {
	status  string
	overlay *ui.ModalOverlay
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

// String implements flag.Value: empty when -seed was not passed, else the decimal seed.
func (o *optionalInt64Flag) String() string {
	if !o.set {
		return ""
	}
	return strconv.FormatInt(o.value, 10)
}

// Set implements flag.Value, parsing a base-10 int64 and marking the flag as present.
func (o *optionalInt64Flag) Set(s string) error {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	o.value = v
	o.set = true
	return nil
}

// resolveProceduralSeed returns the explicit -seed value, or 0 under tests, or a time-based seed.
func resolveProceduralSeed(f *optionalInt64Flag) int64 {
	if f.set {
		return f.value
	}
	if testing.Testing() {
		return 0
	}
	return time.Now().UnixNano()
}

// main parses flags, opens the tcell screen, runs the input/render loop (HUD, help, modals,
// optional fire animation), and tears down on quit.
func main() {
	startLives := flag.Int("lives", 3, "starting lives per level (use -1 for unlimited)")
	seedFlag := &optionalInt64Flag{}
	flag.Var(seedFlag, "seed", "base RNG seed for procedural levels (omit for random from clock; -seed 0 fixes zero)")
	flag.Parse()

	seed := resolveProceduralSeed(seedFlag)
	pack, err := loadPack(seed)
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
	var modal *ui.ModalOverlay
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
		ui.DrawStr(s, 0, lineY, sw, fmt.Sprintf(" %s", g.LevelName), titleSt)
		lineY++
		if lineY < sh {
			livesStr := ui.FormatLives(g.Lives, *startLives)
			left := fmt.Sprintf(" Lives: %s   Cells: %d", livesStr, g.Board.NonEmptyCount())
			ui.DrawStr(s, 0, lineY, sw, left, msgSt)
		}
		lineY++
		if lineY < sh && status != "" {
			st := msgSt
			if strings.HasPrefix(status, "Blocked") {
				st = blockedSt
			} else if strings.HasPrefix(status, "Cleared") {
				st = winSt
			}
			ui.DrawStr(s, 0, lineY, sw, " "+status, st)
		}
		lineY++
		if lineY < sh {
			ui.DrawStr(s, 0, lineY, sw, " hjkl/←↑↓→ move  space/enter fire  r restart  n/p level  ? help  q quit", helpSt)
		}

		if showHelp {
			ui.DrawHelpOverlay(s, sw, sh, base)
		}
		if modal != nil {
			ui.DrawModalOverlay(s, sw, sh, modal, def)
		}
		if generatingN > 0 {
			ui.DrawGeneratingOverlay(s, sw, sh, generatingN, def)
		}
		s.Show()
	}
	tryStartOrApplyFire := func(x, y int) {
		started := tryStartFireAnimation(g, x, y, &anim, animStep)
		if !started {
			fr := applyFire(g, x, y, *startLives)
			status, modal = fr.status, fr.overlay
		}
	}
	interruptAndFire := func(x, y int) {
		if anim.active {
			fr := applyFire(g, anim.fireX, anim.fireY, *startLives)
			status, modal = fr.status, fr.overlay
			anim.active = false
		}
		if g.Won() || g.Lost() {
			return
		}
		tryStartOrApplyFire(x, y)
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
				interruptAndFire(cx, cy)
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
					interruptAndFire(cx, cy)
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

// loadPack builds a procedural pack and ensures level 0 can be generated (fail fast).
func loadPack(seed int64) (*levels.Pack, error) {
	p := levels.NewProceduralPack(seed)
	if _, _, err := p.LevelAt(0); err != nil {
		return nil, err
	}
	return p, nil
}

// newGameForLevel loads level idx; negative startLives is mapped to a huge life pool for “unlimited”.
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

// newGameWithGenOverlay shows a “generating” overlay if the procedural level is not cached yet.
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

// resetLevel reloads the same index from the pack and resets lives (replay current level).
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

// clampCursor keeps the cursor inside the board rectangle.
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

// moveCursor applies a delta in logical cells and clamps to the board.
func moveCursor(g *game.Game, cx, cy *int, dx, dy int) {
	*cx += dx
	*cy += dy
	clampCursor(g, cx, cy)
}

// applyFire runs TryFire and maps outcomes to status text and win/lose modals (no-op if already terminal).
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
				overlay: &ui.ModalOverlay{
					Positive: true,
					Lines: []string{
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
				overlay: &ui.ModalOverlay{
					Positive: false,
					Lines: []string{
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

// tryStartFireAnimation prepares ray-snake frames for a clearing shot; returns false if animation is skipped.
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

// buildPointerFrames builds per-step overlays: head slides along ray then past the edge while the tail follows.
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

// headPositionForStep is the animated head cell after step steps (1-based): along ray then off-board.
func headPositionForStep(ray []struct{ X, Y int }, dx, dy, step int) (int, int) {
	if step <= len(ray) {
		p := ray[step-1]
		return p.X, p.Y
	}
	last := ray[len(ray)-1]
	extra := step - len(ray)
	return last.X + extra*dx, last.Y + extra*dy
}

// straightBodyRune is the wire rune left behind as the head moves (matches fire axis).
func straightBodyRune(d game.Direction) rune {
	switch d {
	case game.North, game.South:
		return '│'
	default:
		return '─'
	}
}

// fireTravelCells lists empty cells from the head cell along the open ray to the board edge (exclusive of head).
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

