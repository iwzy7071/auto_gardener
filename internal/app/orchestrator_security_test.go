package app

import (
	"path/filepath"
	"testing"
)

func TestTreeGoalSpecRejectsGoalPathOutsideTreeDir(t *testing.T) {
	dataDir := t.TempDir()
	orch := &Orchestrator{dataDir: dataDir}
	task := &Task{ID: "forest_abc", Title: "Task"}
	tr := &Tree{ID: "tree_abc", TaskID: task.ID, GoalPath: filepath.Join(dataDir, "outside", "goal.md")}

	goal := orch.treeGoalSpec(task, tr)
	want := filepath.Join(dataDir, "forests", task.ID, "trees", tr.ID, "goal.md")
	if filepath.Clean(goal.Path) != filepath.Clean(want) {
		t.Fatalf("goal path = %q, want %q", goal.Path, want)
	}
}
