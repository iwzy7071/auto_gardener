package app

import (
	"strings"
	"testing"

	"auto_gardener/internal/codex"
)

func TestCreateTaskRejectsOversizedPrompt(t *testing.T) {
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(store, codex.MockRunner{}, store.DataDir(), "")

	_, err = orch.CreateTask(strings.Repeat("目", maxCreateTaskPromptRunes+1), "")
	if err == nil || !strings.Contains(err.Error(), "任务内容过长") {
		t.Fatalf("expected oversized prompt error, got %v", err)
	}
	if tasks := store.ListTasks(); len(tasks) != 0 {
		t.Fatalf("oversized prompt should not create a task: %+v", tasks)
	}
}
