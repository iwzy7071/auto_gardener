package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestAppendGardenerLogRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	dataDir := t.TempDir()
	forestDir := filepath.Join(dataDir, "forests", "forest_abc", "gardener")
	if err := os.MkdirAll(forestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dataDir, "outside.log")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(forestDir, "log.md")
	if err := os.Symlink(target, logPath); err != nil {
		t.Fatal(err)
	}
	store := &Store{tasks: map[string]*Task{"forest_abc": {
		ID: "forest_abc", LogPath: logPath, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}}, events: NewEventHub()}

	store.AppendGardenerLog("forest_abc", "new progress")
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("symlink target was appended: %q", data)
	}
	task, ok := store.GetTask("forest_abc")
	if !ok || len(task.GardenerProgress) == 0 {
		t.Fatalf("in-memory progress should still update: %#v", task)
	}
}
