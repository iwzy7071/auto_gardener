package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreLoadLimitsStartupTreesPerTask(t *testing.T) {
	dataDir := t.TempDir()
	forestDir := filepath.Join(dataDir, "forests", "forest_test")
	treesRoot := filepath.Join(forestDir, "trees")
	if err := os.MkdirAll(treesRoot, 0700); err != nil {
		t.Fatal(err)
	}
	taskPayload, err := json.Marshal(Task{ID: "forest_test", Title: "test", CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forestDir, "forest.json"), taskPayload, 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxStartupTreesPerTask+25; i++ {
		id := fmt.Sprintf("tree_%04d", i)
		dir := filepath.Join(treesRoot, id)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		payload, err := json.Marshal(Tree{ID: id, TaskID: "forest_test", Name: id})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "tree.json"), payload, 0600); err != nil {
			t.Fatal(err)
		}
	}

	store, err := NewStore(dataDir, NewEventHub())
	if err != nil {
		t.Fatal(err)
	}
	task, ok := store.GetTask("forest_test")
	if !ok {
		t.Fatal("expected task to load")
	}
	if got := len(task.Trees); got != maxStartupTreesPerTask {
		t.Fatalf("loaded trees = %d, want %d", got, maxStartupTreesPerTask)
	}
}
