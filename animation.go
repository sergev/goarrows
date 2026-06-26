package main

import (
	"time"

	"goarrows/game"
	"goarrows/ui"
)

type animState struct {
	active   bool
	hidePath []game.Point         // original fired path (masked during animation)
	frames   []ui.FireAnimOverlay // precomputed snake frames
	step     int
	nextStep time.Time
	fireX    int
	fireY    int
}

// tryStartFireAnimation prepares ray-snake frames for a clearing shot; returns false if animation is skipped.
func tryStartFireAnimation(g *game.Game, cx, cy int, anim *animState, stepDur time.Duration) bool {
	if g.Won() || g.Lost() || !g.Board.InBounds(cx, cy) {
		return false
	}
	c := g.Board.At(cx, cy)
	if c.IsEmpty() || !c.IsHead() || !game.RayEscapes(g.Board, cx, cy) {
	}
	path, err := game.PathFromHead(g.Board, cx, cy)
	if err != nil || len(path) == 0 {
		return false
	}
	// An edge head has no travel cells (the escape ray is empty); the body must still
	// slide off, so we do not bail here. BuildFireFrames handles an empty ray.
	frames, ok := ui.BuildFireFrames(g.Board, path, c.R)
	if !ok || len(frames) == 0 {
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
