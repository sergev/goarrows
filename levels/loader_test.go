package levels

import "testing"

func TestLoadEmbedded(t *testing.T) {
	p, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Boards) == 0 || len(p.Names) != len(p.Boards) {
		t.Fatalf("pack: %d boards, %d names", len(p.Boards), len(p.Names))
	}
}
