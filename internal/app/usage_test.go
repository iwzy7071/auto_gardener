package app

import (
	"fmt"
	"testing"
	"time"
)

func TestAllUsageLimitsAggregateTaskCount(t *testing.T) {
	store := &Store{tasks: make(map[string]*Task), dataDir: t.TempDir()}
	for i := 0; i < maxAggregateUsageTasks+5; i++ {
		id := fmt.Sprintf("task-%03d", i)
		store.tasks[id] = &Task{ID: id, CreatedAt: time.Unix(int64(i), 0)}
	}

	usage := store.AllUsage()
	if len(usage) != maxAggregateUsageTasks {
		t.Fatalf("expected %d usage summaries, got %d", maxAggregateUsageTasks, len(usage))
	}
	if usage[0].TaskID != "task-204" {
		t.Fatalf("expected newest task first, got %s", usage[0].TaskID)
	}
}
