package app

import "testing"

func TestTrimStoredMessagesKeepsRecentMessages(t *testing.T) {
	task := &Task{}
	for i := 0; i < maxStoredMessages+10; i++ {
		task.Messages = append(task.Messages, Message{ID: string(rune('a' + i%26))})
	}
	oldFirst := task.Messages[10].ID
	trimStoredMessages(task)
	if len(task.Messages) != maxStoredMessages {
		t.Fatalf("message count = %d, want %d", len(task.Messages), maxStoredMessages)
	}
	if task.Messages[0].ID != oldFirst {
		t.Fatalf("oldest retained message = %q, want %q", task.Messages[0].ID, oldFirst)
	}
}
