package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreLoadLimitsStartupTaskDirs(t *testing.T) {
	dataDir := t.TempDir()
	root := filepath.Join(dataDir, "forests")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxStartupTaskDirs+25; i++ {
		id := fmt.Sprintf("forest_%04d", i)
		dir := filepath.Join(root, id)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		payload, err := json.Marshal(Task{ID: id, Title: id, CreatedAt: time.Now()})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "forest.json"), payload, 0600); err != nil {
			t.Fatal(err)
		}
	}

	store, err := NewStore(dataDir, NewEventHub())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(store.ListTasks()); got != maxStartupTaskDirs {
		t.Fatalf("loaded tasks = %d, want %d", got, maxStartupTaskDirs)
	}
}
