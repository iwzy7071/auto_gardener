package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReadJSONRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "outside.json")
	if err := os.WriteFile(target, []byte(`{"secret":"value"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "state.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	var out map[string]string
	err := readJSON(link, &out)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink JSON read to be rejected, got %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("symlink JSON content should not be decoded: %#v", out)
	}
}
