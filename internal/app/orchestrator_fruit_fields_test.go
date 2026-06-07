package app

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestWriteFruitLimitsReportFields(t *testing.T) {
	orch := &Orchestrator{dataDir: t.TempDir()}
	long := strings.Repeat("x", maxFruitReportFieldRunes+300)
	task := &Task{ID: "forest_test", Title: long, WorkspacePath: long, ScratchPath: long}
	tr := &Tree{ID: "tree_test", TaskID: task.ID, Name: long, Objective: long, Scope: []string{long}}

	path, err := orch.writeFruit(task, tr, "short output", fmt.Errorf(long), time.Now(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if strings.Contains(body, long) {
		t.Fatalf("fruit report contains unbounded field")
	}
	if !strings.Contains(body, strings.Repeat("x", maxFruitReportFieldRunes)+"...") {
		t.Fatalf("fruit report does not contain truncated field marker")
	}
}
