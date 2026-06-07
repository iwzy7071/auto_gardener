package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestAppendTreeProgressRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	dataDir := t.TempDir()
	treeDir := filepath.Join(dataDir, "forests", "forest_abc", "trees", "tree_abc")
	if err := os.MkdirAll(treeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dataDir, "outside-progress.log")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	progressPath := filepath.Join(treeDir, "progress.log")
	if err := os.Symlink(target, progressPath); err != nil {
		t.Fatal(err)
	}
	store := &Store{dataDir: dataDir, events: NewEventHub(), tasks: map[string]*Task{"forest_abc": {
		ID: "forest_abc", CreatedAt: time.Now(), UpdatedAt: time.Now(), Trees: []*Tree{{ID: "tree_abc", TaskID: "forest_abc"}},
	}}}

	store.AppendTreeProgress("forest_abc", "tree_abc", "new progress")
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("symlink target was appended: %q", data)
	}
	task, ok := store.GetTask("forest_abc")
	if !ok || len(task.Trees) != 1 || len(task.Trees[0].Progress) == 0 {
		t.Fatalf("in-memory tree progress should still update: %#v", task)
	}
}
