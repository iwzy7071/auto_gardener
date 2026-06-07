package app

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestWriteTreeGoalLimitsModelControlledFields(t *testing.T) {
	orch := &Orchestrator{dataDir: t.TempDir()}
	long := strings.Repeat("x", maxGoalReportFieldRunes+500)
	task := &Task{ID: "forest_test", Title: long, WorkspacePath: long}
	tr := &Tree{ID: "tree_test", TaskID: task.ID, Name: long, Objective: long, Scope: []string{long}}
	tr.GoalPath = orch.treeGoalPath(task.ID, tr.ID)

	if err := orch.writeTreeGoal(task, tr, long, time.Now(), nil, long, ""); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(tr.GoalPath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if strings.Contains(body, long) {
		t.Fatalf("goal report contains unbounded field")
	}
	if !strings.Contains(body, strings.Repeat("x", maxGoalReportFieldRunes)+"...") {
		t.Fatalf("goal report does not contain truncated field marker")
	}
}
