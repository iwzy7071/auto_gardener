package app

import (
	"strings"
	"testing"

	"auto_gardener/internal/codex"
)

func TestSendMessageRejectsOversizedContent(t *testing.T) {
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	newTestTask(t, store, "forest_message_limit")
	orch := NewOrchestrator(store, codex.MockRunner{}, store.DataDir(), "")

	_, err = orch.SendMessage("forest_message_limit", strings.Repeat("长", maxSendMessageContentRunes+1))
	if err == nil || !strings.Contains(err.Error(), "消息过长") {
		t.Fatalf("expected oversized message error, got %v", err)
	}

	task, ok := store.GetTask("forest_message_limit")
	if !ok {
		t.Fatal("task missing")
	}
	if len(task.Messages) != 0 {
		t.Fatalf("oversized message should not be persisted: %+v", task.Messages)
	}
}
