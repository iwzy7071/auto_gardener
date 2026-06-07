package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreLoadRejectsMismatchedTaskID(t *testing.T) {
	dataDir := t.TempDir()
	forestDir := filepath.Join(dataDir, "forests", "forest_safe")
	if err := os.MkdirAll(forestDir, 0700); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(Task{ID: "../escape", Title: "bad", CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forestDir, "forest.json"), payload, 0600); err != nil {
		t.Fatal(err)
	}

	store, err := NewStore(dataDir, NewEventHub())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := store.GetTask("../escape"); ok {
		t.Fatal("loaded task with path traversal ID")
	}
	if _, ok := store.GetTask("forest_safe"); ok {
		t.Fatal("loaded task whose JSON ID did not match directory name")
	}
}

func TestStoreLoadRejectsMismatchedTreeID(t *testing.T) {
	dataDir := t.TempDir()
	forestDir := filepath.Join(dataDir, "forests", "forest_safe")
	treeDir := filepath.Join(forestDir, "trees", "tree_safe")
	if err := os.MkdirAll(treeDir, 0700); err != nil {
		t.Fatal(err)
	}
	task := Task{ID: "forest_safe", Title: "ok", CreatedAt: time.Now(), LogPath: filepath.Join(forestDir, "gardener", "log.md")}
	taskPayload, err := json.Marshal(task)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forestDir, "forest.json"), taskPayload, 0600); err != nil {
		t.Fatal(err)
	}
	treePayload, err := json.Marshal(Tree{ID: "../../escape", TaskID: "forest_safe", Name: "bad"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(treeDir, "tree.json"), treePayload, 0600); err != nil {
		t.Fatal(err)
	}

	store, err := NewStore(dataDir, NewEventHub())
	if err != nil {
		t.Fatal(err)
	}
	loaded, ok := store.GetTask("forest_safe")
	if !ok {
		t.Fatal("expected safe task to load")
	}
	if len(loaded.Trees) != 0 {
		t.Fatalf("loaded mismatched tree ID: %+v", loaded.Trees)
	}
}
