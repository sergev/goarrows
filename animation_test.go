package main

import (
	"testing"
	"time"

	"goarrows/game"
	"goarrows/ui"
)

func TestBuildPointerFrames_EastStraight(t *testing.T) {
	b := game.NewBoard(6, 3)
	b.Set(3, 1, game.Cell{R: '>'})
	b.Set(2, 1, game.Cell{R: '─'})
	b.Set(1, 1, game.Cell{R: '─'})

	path, err := game.PathFromHead(b, 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	ray := fireTravelCells(b, 3, 1)
	frames, ok := buildPointerFrames(b, path, ray, '>')
	if !ok {
		t.Fatal("buildPointerFrames returned !ok")
	}
	if len(frames) != 4 {
		t.Fatalf("frame count got %d want 4", len(frames))
	}

	assertFrameCells(t, frames[0].Cells, []ui.OverlayCell{
		{X: 4, Y: 1, R: '>'},
		{X: 3, Y: 1, R: '─'},
		{X: 2, Y: 1, R: '─'},
	})
	assertFrameCells(t, frames[1].Cells, []ui.OverlayCell{
		{X: 5, Y: 1, R: '>'},
		{X: 4, Y: 1, R: '─'},
		{X: 3, Y: 1, R: '─'},
	})
	assertFrameCells(t, frames[2].Cells, []ui.OverlayCell{
		{X: 6, Y: 1, R: '>'},
		{X: 5, Y: 1, R: '─'},
		{X: 4, Y: 1, R: '─'},
	})
	assertFrameCells(t, frames[3].Cells, []ui.OverlayCell{
		{X: 7, Y: 1, R: '>'},
		{X: 6, Y: 1, R: '─'},
		{X: 5, Y: 1, R: '─'},
	})
}

func TestBuildPointerFrames_NorthStraight(t *testing.T) {
	b := game.NewBoard(4, 6)
	b.Set(2, 2, game.Cell{R: '^'})
	b.Set(2, 3, game.Cell{R: '│'})
	b.Set(2, 4, game.Cell{R: '│'})

	path, err := game.PathFromHead(b, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	ray := fireTravelCells(b, 2, 2)
	frames, ok := buildPointerFrames(b, path, ray, '^')
	if !ok {
		t.Fatal("buildPointerFrames returned !ok")
	}
	if len(frames) != 4 {
		t.Fatalf("frame count got %d want 4", len(frames))
	}
	assertFrameCells(t, frames[0].Cells, []ui.OverlayCell{
		{X: 2, Y: 1, R: '^'},
		{X: 2, Y: 2, R: '│'},
		{X: 2, Y: 3, R: '│'},
	})
	assertFrameCells(t, frames[1].Cells, []ui.OverlayCell{
		{X: 2, Y: 0, R: '^'},
		{X: 2, Y: 1, R: '│'},
		{X: 2, Y: 2, R: '│'},
	})
}

func TestBuildPointerFrames_BentPath(t *testing.T) {
	b := game.NewBoard(6, 4)
	b.Set(3, 1, game.Cell{R: '>'})
	b.Set(2, 1, game.Cell{R: '┌'})
	b.Set(2, 2, game.Cell{R: '┘'})
	b.Set(1, 2, game.Cell{R: '─'})

	path, err := game.PathFromHead(b, 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	ray := fireTravelCells(b, 3, 1)
	frames, ok := buildPointerFrames(b, path, ray, '>')
	if !ok {
		t.Fatal("buildPointerFrames returned !ok")
	}
	if len(frames) != 5 {
		t.Fatalf("frame count got %d want 5", len(frames))
	}
	assertFrameCells(t, frames[0].Cells, []ui.OverlayCell{
		{X: 4, Y: 1, R: '>'},
		{X: 3, Y: 1, R: '─'},
		{X: 2, Y: 1, R: '┌'},
		{X: 2, Y: 2, R: '┘'},
	})
	assertFrameCells(t, frames[1].Cells, []ui.OverlayCell{
		{X: 5, Y: 1, R: '>'},
		{X: 4, Y: 1, R: '─'},
		{X: 3, Y: 1, R: '─'},
		{X: 2, Y: 1, R: '┌'},
	})
}

func TestHeadPositionForStep(t *testing.T) {
	ray := []struct{ X, Y int }{{2, 1}, {3, 1}}
	x, y := headPositionForStep(ray, 1, 0, 1)
	if x != 2 || y != 1 {
		t.Fatalf("step1 got (%d,%d) want (2,1)", x, y)
	}
	x, y = headPositionForStep(ray, 1, 0, 3)
	if x != 4 || y != 1 {
		t.Fatalf("step3 got (%d,%d) want (4,1)", x, y)
	}
}

func TestFireTravelCells_AllDirections(t *testing.T) {
	tests := []struct {
		name string
		w, h int
		hx, hy int
		rune rune
		want []struct{ X, Y int }
	}{
		{"east", 6, 3, 2, 1, '>', []struct{ X, Y int }{{3, 1}, {4, 1}, {5, 1}}},
		{"west", 6, 3, 3, 1, '<', []struct{ X, Y int }{{2, 1}, {1, 1}, {0, 1}}},
		{"north", 4, 6, 2, 3, '^', []struct{ X, Y int }{{2, 2}, {2, 1}, {2, 0}}},
		{"south", 4, 6, 1, 2, 'v', []struct{ X, Y int }{{1, 3}, {1, 4}, {1, 5}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := game.NewBoard(tc.w, tc.h)
			b.Set(tc.hx, tc.hy, game.Cell{R: tc.rune})
			got := fireTravelCells(b, tc.hx, tc.hy)
			assertPoints(t, got, tc.want)
		})
	}
}

func TestTryStartFireAnimation_InitializesState(t *testing.T) {
	b := game.NewBoard(6, 3)
	b.Set(3, 1, game.Cell{R: '>'})
	b.Set(2, 1, game.Cell{R: '─'})
	b.Set(1, 1, game.Cell{R: '─'})
	g := game.NewGame(b, 3, "t")
	var anim animState
	ok := tryStartFireAnimation(g, 3, 1, &anim, 10*time.Millisecond)
	if !ok {
		t.Fatal("expected animation to start")
	}
	if !anim.active {
		t.Fatal("anim.active = false")
	}
	if len(anim.frames) == 0 {
		t.Fatal("anim.frames empty")
	}
	if anim.fireX != 3 || anim.fireY != 1 {
		t.Fatalf("fire origin got (%d,%d) want (3,1)", anim.fireX, anim.fireY)
	}
}

func assertFrameCells(t *testing.T, got, want []ui.OverlayCell) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("cell len got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("cell[%d] got (%d,%d,%q) want (%d,%d,%q)",
				i, got[i].X, got[i].Y, got[i].R, want[i].X, want[i].Y, want[i].R)
		}
	}
}

func assertPoints(t *testing.T, got, want []struct{ X, Y int }) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("point len got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("point[%d] got (%d,%d) want (%d,%d)", i, got[i].X, got[i].Y, want[i].X, want[i].Y)
		}
	}
}
