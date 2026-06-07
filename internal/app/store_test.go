package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadIgnoresLogPathOutsideForest(t *testing.T) {
	dataDir := t.TempDir()
	forestDir := filepath.Join(dataDir, "forests", "forest_abc")
	if err := os.MkdirAll(forestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(dataDir, "outside.log")
	line := fmt.Sprintf("[%s] external secret progress\n", time.Now().Format(time.RFC3339))
	if err := os.WriteFile(outside, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	forestJSON := fmt.Sprintf(`{"id":"forest_abc","status":"Running","gardenerStatus":"Running","logPath":%q}`, outside)
	if err := os.WriteFile(filepath.Join(forestDir, "forest.json"), []byte(forestJSON), 0o644); err != nil {
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
	if len(task.GardenerProgress) != 0 {
		t.Fatalf("outside log progress should not be loaded: %#v", task.GardenerProgress)
	}
}
