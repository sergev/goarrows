package main

import (
	"testing"

	"goarrows/levels"
)

// TestResolveProceduralSeed_unsetUnderTest checks that an unset -seed resolves to 0 when tests run.
func TestResolveProceduralSeed_unsetUnderTest(t *testing.T) {
	f := &optionalInt64Flag{}
	if got := resolveProceduralSeed(f); got != 0 {
		t.Fatalf("unset under test: got %d want 0", got)
	}
}

// TestResolveProceduralSeed_explicit checks that a set flag value is returned verbatim.
func TestResolveProceduralSeed_explicit(t *testing.T) {
	f := &optionalInt64Flag{}
	if err := f.Set("42"); err != nil {
		t.Fatal(err)
	}
	if got := resolveProceduralSeed(f); got != 42 {
		t.Fatalf("got %d want 42", got)
	}
}

// TestLoadPack_procedural ensures loadPack succeeds and yields a procedural pack of expected Len.
func TestLoadPack_procedural(t *testing.T) {
	p, err := loadPack(42)
	if err != nil {
		t.Fatalf("loadPack returned error: %v", err)
	}
	if p == nil {
		t.Fatal("loadPack returned nil pack")
	}
	if got, want := p.Len(), levels.ProceduralLevelCount; got != want {
		t.Fatalf("pack len=%d want %d", got, want)
	}
}
