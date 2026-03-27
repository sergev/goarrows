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
- `-level PATH` — Load a single level file instead of the embedded pack. Levels are newline-separated rows of equal width; see `levels/data/*.txt` for examples.

## Project layout

| Package | Role |
|---------|------|
| `main` | [tcell](https://github.com/gdamore/tcell) screen setup, input loop, HUD (level name, lives, cell count), status messages, and help overlay. |
| `game` | Board model (`Board`, `Cell`), level parsing (`ParseLevel` / `ParseLevelString`), port-based adjacency for wires and heads, validation (full grid, degree constraints, one head per component), tracing a path from a head (`PathFromHead`), and fire / win / lose rules (`TryFire`, `RayEscapes`). |
| `levels` | Embedded `.txt` levels under `levels/data/` (`go:embed`), sorted load order, plus `LoadFile` for a custom path. |
| `ui` | One character per logical cell: `DrawGrid` and display runes. |

The game logic stays independent of the terminal: `TryFire` updates the board and lives; `main` only handles presentation and input.

## Status

This repo is a **playable skeleton**: core rules, a few sample levels, and a minimal TUI are in place; you can extend it with more levels, polish, or features on top of the same `game` package.
