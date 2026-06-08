package app

import (
	"strings"
	"testing"
)

func TestValidAPITaskIDRejectsInvalidIDs(t *testing.T) {
	for _, id := range []string{"", "../forest_abc", "forest/abc", "forest abc", strings.Repeat("a", maxAPITaskIDLength+1)} {
		if validAPITaskID(id) {
			t.Fatalf("validAPITaskID accepted invalid id %q", id)
		}
	}
	if !validAPITaskID("forest_abcdef123456") {
		t.Fatal("validAPITaskID rejected generated task id")
	}
}
