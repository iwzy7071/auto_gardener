package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestWriteTreeGoalRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	dataDir := t.TempDir()
	treeDir := filepath.Join(dataDir, "forests", "forest_abc", "trees", "tree_abc")
	if err := os.MkdirAll(treeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dataDir, "outside-goal.md")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	goalPath := filepath.Join(treeDir, "goal.md")
	if err := os.Symlink(target, goalPath); err != nil {
		t.Fatal(err)
	}

	orch := &Orchestrator{dataDir: dataDir}
	task := &Task{ID: "forest_abc", Title: "Task"}
	tr := &Tree{ID: "tree_abc", TaskID: "forest_abc", GoalPath: goalPath}
	err := orch.writeTreeGoal(task, tr, "Running", time.Now(), nil, "note", "")
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink goal write to be rejected, got %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("symlink target was overwritten: %q", data)
	}
}
