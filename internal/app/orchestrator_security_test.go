package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestWriteFruitRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	dataDir := t.TempDir()
	treeDir := filepath.Join(dataDir, "forests", "forest_abc", "trees", "tree_abc")
	if err := os.MkdirAll(treeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dataDir, "outside-fruit.md")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(treeDir, "fruit.md")); err != nil {
		t.Fatal(err)
	}

	orch := &Orchestrator{dataDir: dataDir}
	_, err := orch.writeFruit(&Task{ID: "forest_abc", Title: "Task"}, &Tree{ID: "tree_abc", TaskID: "forest_abc"}, "output", nil, time.Now(), time.Now())
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink fruit write to be rejected, got %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("symlink target was overwritten: %q", data)
	}
}
