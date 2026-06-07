package app

import (
	"strings"
	"testing"
)

func TestValidWorkspaceFileTreeFilterRejectsInvalidIDs(t *testing.T) {
	for _, id := range []string{"tree/abc", "tree abc", strings.Repeat("a", maxWorkspaceFileTreeFilterLength+1)} {
		if validWorkspaceFileTreeFilter(id) {
			t.Fatalf("validWorkspaceFileTreeFilter accepted invalid id %q", id)
		}
	}
	for _, id := range []string{"", "tree_abcdef123456"} {
		if !validWorkspaceFileTreeFilter(id) {
			t.Fatalf("validWorkspaceFileTreeFilter rejected valid id %q", id)
		}
	}
}
