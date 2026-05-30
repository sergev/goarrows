# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build, run, and test

- Run the game: `go run .` (requires a UTF-8 terminal).
- Build: `go build` (or `make`).
- Full test suite: `make test` (uses `gotestsum`, auto-installs if missing) or `go test -timeout 10s ./...`. Always keep `-timeout 10s` so the suite stays bounded.
- Single test: `go test -run TestName -timeout 10s ./game` (replace package as needed).
- Coverage: `make cover`.
- Runtime flags: `-lives N` (default 3, `-1` for unlimited); `-seed N` for a deterministic level sequence (omit for a clock-based base seed).

## Architecture

Three packages with a strict one-way dependency: `main` → `ui` → `game`. The game rules stay independent of the terminal.

- **`game`** — pure logic, no `tcell`/terminal imports. Owns `Board`, `Cell`, `Direction`, port bitmask, level parsing, validation, firing (`TryFire`, `RayEscapes`, `PathFromHead`), procedural generation (`GenerateBoard`/`GenGrow`), and the on-demand `Levels` generator (`NewLevels(seed)` + `Levels.At`, which generates lazily and memoizes per seed).
- **`ui`** — renders a `game.Board` to a `tcell.Screen`. Maps logical cell `(x, y)` to screen column `2*x` (row `y`); `GridSize` is `(2*w-1, h)` because horizontal wires get joined with `─`. Also owns HUD, help/modal/generating overlays, and **builds** fire-animation frames (`BuildFireFrames`).
- **`main`** — tcell setup, input loop, HUD layout, status/overlay state, and fire-animation **timing/stepping** (it requests frames from `ui.BuildFireFrames`, then steps them). Must not contain game rules.

### One responsibility per file

Add new code to the matching file rather than growing a catch-all.

- `game`: `board.go` (model), `game.go` (`Game`, `TryFire`, `RayEscapes`), `ports.go` (port bitmask, `EffectivePorts`, `linked`, `directionFromTo`), `validate.go`, `path.go`, `level.go` (parsing), `gen.go` / `gen_reverse.go` / `paint.go` (generation), `levels.go` (on-demand generator: `NewLevels`, `At`, `Count`, `SideLen`, `Ready`, `levelRNG`), `solvable.go` (`VerifyGreedyFirstClearsBoard`, `VerifySolvable`, `VerifySolvableFast`).
- `ui`: `grid.go` (`DrawGrid`, `GridSize`), `overlay.go` (HUD/modals), `animation.go` (`BuildFireFrames`, `buildPointerFrames`, `fireTravelCells`).
- `main` (root): `main.go` (event/render loop), `flags.go` (CLI + `resolveProceduralSeed`), `levels.go` (level/game glue), `cursor.go`, `fire.go` (fire outcome → status/modal), `animation.go` (`animState`, `tryStartFireAnimation`).

### Level generation

Level *k* is a `(k+2)x(k+2)` grid. The generator (`generateFullBoardReverse` in `gen_reverse.go`) places arrows one at a time in **reverse firing order** — the first arrow placed will fire last; the last placed will fire first. Each new head is positioned so its open firing ray is clear of all already-placed cells, and its body is grown by a length-capped random walk that avoids the head's own ray and any extension that would form an accidental cross-path port link. Forward play then clears every arrow by construction. `ValidatePartialBoard`, `VerifySolvableFast`, and `VerifyGreedyFirstClearsBoard` are test-only assertions of this invariant; the generation hot path runs only a `growPlayfulEnough` check (≤ half the heads fireable at start) and restarts on failure with bounded retries.

## Conventions

- **Coordinates are row-major**: iterate `y` then `x`, index flat slices as `y*W + x`. Always `Board.InBounds` before `At`/`Set`.
- **Cells**: `Cell.R == 0` means empty. Wires are `─│┌┐└┘`. Heads are normalized to `▲▼◀▶`; ASCII `^ v V < >` are accepted on input and normalized via `normalizeHeadRune`.
- **Movement**: use the `Direction` enum (`North/East/South/West`) with `Delta(d)` — never hardcode `dx/dy` literals. Use `oppositeDir` rather than re-deriving.
- **RNG**: stdlib `math/rand/v2` (PCG generators) only — never the global `math/rand`. Thread an explicit `*rand.Rand` through generation; derive per-level seeds via `levelRNG(seed, idx)`.
- **Errors vs. panics**: library code in `game` returns `error` for invalid boards/sizes. Panics are only acceptable in `main` for unrecoverable setup (e.g. a cached level failing to reload).
- **Dependencies**: the only external dep is `github.com/gdamore/tcell/v2`. Don't add new deps without a strong reason.

## Board invariants (preserve these)

- A head has exactly one body link (one effective port).
- A wire cell has degree 1 (tail) or 2 (internal).
- Each connected component contains exactly one head.
- Adjacency requires mutual consent (both cells expose the shared edge); two heads never link.

## Testing notes

- Build board fixtures inline with `ParseLevelString`; for intentionally invalid layouts use the no-validate helper (`boardFromLinesNoValidate` in tests).
- Prefer table-driven tests.
- Tests rely on deterministic seeds: `resolveProceduralSeed` returns `0` under `testing.Testing()`.
