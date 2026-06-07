package app

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestWriteFruitLimitsFullCLIOutput(t *testing.T) {
	orch := &Orchestrator{dataDir: t.TempDir()}
	task := &Task{ID: "forest_test", Title: "test", WorkspacePath: t.TempDir(), ScratchPath: t.TempDir()}
	tr := &Tree{ID: "tree_test", TaskID: task.ID, Name: "test tree", Objective: "test objective"}
	output := strings.Repeat("x", maxFruitOutputRunes+500)

	path, err := orch.writeFruit(task, tr, output, nil, time.Now(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if strings.Contains(body, output) {
		t.Fatalf("fruit report contains unbounded full CLI output")
	}
	if !strings.Contains(body, strings.Repeat("x", maxFruitOutputRunes)+"...") {
		t.Fatalf("fruit report does not contain truncated CLI output marker")
	}
}
