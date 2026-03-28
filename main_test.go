package main

import "testing"

func TestResolveProceduralSeed_unsetUnderTest(t *testing.T) {
	f := &optionalInt64Flag{}
	if got := resolveProceduralSeed(f); got != 0 {
		t.Fatalf("unset under test: got %d want 0", got)
	}
}

func TestResolveProceduralSeed_explicit(t *testing.T) {
	f := &optionalInt64Flag{}
	if err := f.Set("42"); err != nil {
		t.Fatal(err)
	}
	if got := resolveProceduralSeed(f); got != 42 {
		t.Fatalf("got %d want 42", got)
	}
}
