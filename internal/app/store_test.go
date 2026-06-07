package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestWriteScheduleRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	dataDir := t.TempDir()
	forestDir := filepath.Join(dataDir, "forests", "forest_abc", "gardener")
	if err := os.MkdirAll(forestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dataDir, "outside.md")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	schedulePath := filepath.Join(forestDir, "schedule.md")
	if err := os.Symlink(target, schedulePath); err != nil {
		t.Fatal(err)
	}
	store := &Store{tasks: map[string]*Task{"forest_abc": {
		ID: "forest_abc", SchedulePath: schedulePath, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}}}

	err := store.WriteSchedule("forest_abc", "overwrite")
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink write to be rejected, got %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("symlink target was overwritten: %q", data)
	}
}
