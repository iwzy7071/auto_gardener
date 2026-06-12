package app

import (
	"strings"
	"testing"
	"time"

	"auto_gardener/internal/codex"
)

func TestAskMessageDoesNotStartSubtasks(t *testing.T) {
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(store, codex.MockRunner{}, store.DataDir(), "")
	workspace := t.TempDir()
	now := time.Now()
	task := &Task{
		ID:             "forest_ask",
		Title:          "Ask task",
		Prompt:         "Answer questions",
		WorkspacePath:  workspace,
		ScratchPath:    workspace,
		Status:         StatusFinished,
		GardenerStatus: StatusFinished,
		CreatedAt:      now,
		UpdatedAt:      now,
		Trees: []*Tree{{
			ID:        "tree_done",
			TaskID:    "forest_ask",
			Name:      "Existing report",
			Status:    StatusFinished,
			FruitPath: "fruit.md",
		}},
		Messages: []Message{{ID: "msg_initial", Role: RoleGardener, Content: "Ready", CreatedAt: now}},
	}
	if err := store.AddTask(task); err != nil {
		t.Fatal(err)
	}

	updated, err := orch.AskMessage(task.ID, "这是什么任务？")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != StatusFinished || updated.GardenerStatus != StatusFinished {
		t.Fatalf("ask changed status to task=%s gardener=%s", updated.Status, updated.GardenerStatus)
	}
	if len(updated.Trees) != 1 {
		t.Fatalf("ask created subtasks; got %d trees", len(updated.Trees))
	}
	if len(updated.Messages) < 3 {
		t.Fatalf("ask did not append user and gardener messages: %+v", updated.Messages)
	}
	last := updated.Messages[len(updated.Messages)-1]
	if last.Role != RoleGardener || !strings.Contains(last.Content, "Mock quick answer") {
		t.Fatalf("last message = %+v, want mock quick answer", last)
	}
}
