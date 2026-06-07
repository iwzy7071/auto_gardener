package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSkipsMismatchedDiskTreeID(t *testing.T) {
	dataDir := t.TempDir()
	forestDir := filepath.Join(dataDir, "forests", "forest_abc")
	treeDir := filepath.Join(forestDir, "trees", "tree_safe")
	if err := os.MkdirAll(treeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forestDir, "forest.json"), []byte(`{"id":"forest_abc","status":"Running","gardenerStatus":"Running"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(treeDir, "tree.json"), []byte(`{"id":"../escape","taskId":"forest_abc","status":"Running"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := NewStore(dataDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	task, ok := store.GetTask("forest_abc")
	if !ok {
		t.Fatal("task was not loaded")
	}
	if len(task.Trees) != 0 {
		t.Fatalf("mismatched tree ID should be skipped, got %#v", task.Trees)
	}
	if _, err := os.Stat(filepath.Join(forestDir, "escape", "progress.log")); !os.IsNotExist(err) {
		t.Fatalf("unexpected write through mismatched tree ID: %v", err)
	}
}
