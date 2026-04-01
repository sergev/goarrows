package levels

import (
	"testing"

	"goarrows/game"
)

func TestInlineFixturesBuildPack(t *testing.T) {
	cases := []struct {
		name  string
		level string
	}{
		{name: "vertical", level: "▲\n│"},
		{name: "horizontal", level: "──▶"},
		{name: "two_tall", level: "▲▲\n││"},
		{name: "elbow", level: "└▶"},
	}

	p := &Pack{
		Names:  make([]string, 0, len(cases)),
		Boards: make([]game.Board, 0, len(cases)),
	}
	for _, tc := range cases {
		b, err := game.ParseLevelString(tc.level)
		if err != nil {
			t.Fatalf("%s: parse failed: %v", tc.name, err)
		}
		if b.W == 0 || b.H == 0 {
			t.Fatalf("%s: empty board parsed", tc.name)
		}
		p.Names = append(p.Names, tc.name)
		p.Boards = append(p.Boards, b)
	}
	if len(p.Boards) == 0 || len(p.Names) != len(p.Boards) {
		t.Fatalf("pack: %d boards, %d names", len(p.Boards), len(p.Names))
	}
}

func TestInlineFixtureParseError(t *testing.T) {
	invalid := "▶\n.."
	if _, err := game.ParseLevelString(invalid); err == nil {
		t.Fatal("expected parse error for invalid fixture")
	}
}
