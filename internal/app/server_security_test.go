package app

import (
	"strings"
	"testing"
)

func TestValidAPITreeIDRejectsInvalidIDs(t *testing.T) {
	for _, id := range []string{"", "../tree_abc", "tree/abc", "tree abc", strings.Repeat("a", maxAPITreeIDLength+1)} {
		if validAPITreeID(id) {
			t.Fatalf("validAPITreeID accepted invalid id %q", id)
		}
	}
	if !validAPITreeID("tree_abcdef123456") {
		t.Fatal("validAPITreeID rejected generated tree id")
	}
}
