# Go Arrows

A small terminal puzzle written in the Go language. The playing field is covered with arrows, some straight, some curved. Your goal is to remove all the arrows one by one. You move the cursor, select an arrowhead, and **shoot**: the arrow slides straight along its flight path. If the beam reaches the edge of the field without hitting another arrow, it is removed. If something blocks the beam, you **lose**.

## Run

Requires a UTF-8 terminal.

```bash
go run .
```

## Controls

| Input | Action |
|--------|--------|
| `h` `j` `k` `l` or arrow keys | Move cursor |
| Space, Enter, or `f` | Fire at the cell under the cursor |
| `r` | Restart current level |
| `n` / `p` | Next / previous level |
| `?` | Toggle help overlay |
| `q` or Ctrl+C | Quit |

After a win or game over, `n`, `p`, `r`, and `q` behave as indicated on the status line.

## Flags

- `-lives N` — Starting lives per level (default `3`). Use `-1` for unlimited.
- `-seed N` — Base seed for procedural level generation. **Omit** `-seed` to pick a random base seed from the system clock (each run differs). Pass `-seed N` for a reproducible sequence. For each level, generation uses `N`, then `N+1`, `N+2`, … (up to a fixed try limit) until a board is built, so a failing draw does not abort the game.

## Procedural levels

By default the game uses a **procedural pack**: level *k* (1-based in the HUD) is a **(k+2)×(k+2)** grid (level 1 → 3×3, then 4×4, 5×5, …). The grow generator seeds `min(n,n)` small arrows and extends them at random until stuck; the board may have **empty cells**. A board is accepted only if at most **half** the arrow heads have a clear shot at the start (so it is not trivially easy), and if **greedy row-major clearing** (repeatedly fire the first head whose ray escapes) removes every arrow. Generation is deterministic for a given base seed when you pass **`-seed`**. Levels are generated on demand and memoized per run.

## Project layout

| Package | Role |
|---------|------|
| `main` | [tcell](https://github.com/gdamore/tcell) screen setup, input loop, HUD (level name, lives, cell count), status messages, and help overlay. |
| `game` | Board model, parsing, validation, `PathFromHead`, `TryFire` / `RayEscapes`, procedural `GenerateBoard` (`GenGrow`, `ValidatePartialBoard`, `VerifyGreedyFirstClearsBoard`), `GenerateFullBoard`, and `VerifySolvable` (backtracking, for tests). |
| `levels` | `NewProceduralPack(seed)` / `Pack.LevelAt` for on-demand boards; tests build fixtures inline. |
| `ui` | `DrawGrid` maps each logical cell to screen column `2*x` (height `y`), inserts `─` between neighbors when `game.HorizontalLink` is true so horizontal wires read as one continuous line; `GridSize` is `(2*w-1, h)`. |

The game logic stays independent of the terminal: `TryFire` updates the board and lives; `main` only handles presentation and input.

## Status

Playable TUI with procedural full-board levels.
