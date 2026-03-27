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
- `-seed N` — 64-bit seed for procedural level generation (default `0`). Same seed reproduces the same sequence of boards.
- `-level PATH` — Load a single level file instead of procedural levels. Levels are newline-separated rows of equal width; see `levels/data/*.txt` for examples.

## Procedural levels

By default the game uses a **procedural pack**: level *k* (1-based in the HUD) is a random full **(k+2)×(k+2)** grid (level 1 → 3×3, then 4×4, 5×5, …). Boards are built in **reverse removal order** so that at every step at least one arrow can legally fire until the grid is clear. Path lengths are randomized within bounds that scale with the side length so snakes are neither tiny nor whole-board by default. Levels are generated on demand and memoized per run.

## Project layout

| Package | Role |
|---------|------|
| `main` | [tcell](https://github.com/gdamore/tcell) screen setup, input loop, HUD (level name, lives, cell count), status messages, and help overlay. |
| `game` | Board model, parsing, validation, `PathFromHead`, `TryFire` / `RayEscapes`, procedural `GenerateFullBoard`, and `VerifySolvable` (backtracking, for tests). |
| `levels` | `NewProceduralPack` / `Pack.LevelAt` for on-demand boards; `LoadFile` for `-level`; `LoadEmbedded` for tests and sample `.txt` under `levels/data/`. |
| `ui` | One character per logical cell: `DrawGrid` and display runes. |

The game logic stays independent of the terminal: `TryFire` updates the board and lives; `main` only handles presentation and input.

## Status

Playable TUI with procedural full-board levels and optional hand-authored `-level` files; sample grids remain under `levels/data/` for reference and tests.
