package app

import (
	"strings"
	"testing"
)

func TestValidWorkspaceFileForestFilterRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"abc", "1/2", "0", strings.Repeat("9", maxWorkspaceFileForestFilterLength+1)} {
		if validWorkspaceFileForestFilter(value) {
			t.Fatalf("validWorkspaceFileForestFilter accepted invalid value %q", value)
		}
	}
	for _, value := range []string{"", "1", "123456"} {
		if !validWorkspaceFileForestFilter(value) {
			t.Fatalf("validWorkspaceFileForestFilter rejected valid value %q", value)
		}
	}
}
