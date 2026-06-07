package app

import "testing"

func TestNewStableUsageIDSeparatesPartBoundaries(t *testing.T) {
	first := newStableUsageID("a\x00b", "c")
	second := newStableUsageID("a", "b\x00c")
	if first == second {
		t.Fatalf("stable usage id collision: %s", first)
	}
}
