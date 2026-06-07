package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadResetsScratchPathOutsideManagedRoots(t *testing.T) {
	dataDir := t.TempDir()
	forestDir := filepath.Join(dataDir, "forests", "forest_abc")
	if err := os.MkdirAll(forestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "attacker-workdir")
	forestJSON := fmt.Sprintf(`{"id":"forest_abc","status":"Running","gardenerStatus":"Running","workspacePath":%q,"scratchPath":%q}`, dataDir, outside)
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
	want := filepath.Join(dataDir, "scratch", "forest_abc")
	if filepath.Clean(task.ScratchPath) != filepath.Clean(want) {
		t.Fatalf("scratch path = %q, want %q", task.ScratchPath, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("managed scratch path was not created: %v", err)
	}
}
