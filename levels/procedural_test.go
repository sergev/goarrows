package levels

import (
	"testing"
)

func TestProceduralPackLevelAt(t *testing.T) {
	p := NewProceduralPack(42)
	b, name, err := p.LevelAt(0)
	if err != nil {
		t.Fatal(err)
	}
	if name == "" || b.W != 3 || b.H != 3 {
		t.Fatalf("level 0: name=%q board=%dx%d", name, b.W, b.H)
	}
	b2, _, err := p.LevelAt(0)
	if err != nil {
		t.Fatal(err)
	}
	if b2.W != b.W || b2.H != b.H {
		t.Fatal("memo mismatch")
	}
	b3, name3, err := p.LevelAt(2)
	if err != nil {
		t.Fatal(err)
	}
	if b3.W != 5 || b3.H != 5 {
		t.Fatalf("level 2 want 5×5 got %d×%d", b3.W, b3.H)
	}
	if name3 == "" {
		t.Fatal("empty name")
	}
	if p.Len() != ProceduralLevelCount {
		t.Fatalf("Len: %d", p.Len())
	}
}
