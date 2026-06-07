package app

import (
	"testing"
	"time"
)

func TestListRunningTasksOnlyClonesRunningTasks(t *testing.T) {
	store := &Store{tasks: map[string]*Task{}}
	store.tasks["running"] = &Task{ID: "running", Status: StatusRunning, CreatedAt: time.Now()}
	store.tasks["finished"] = &Task{ID: "finished", Status: StatusFinished, CreatedAt: time.Now().Add(time.Second)}

	got := store.ListRunningTasks()
	if len(got) != 1 || got[0].ID != "running" {
		t.Fatalf("ListRunningTasks = %+v, want only running task", got)
	}
	got[0].Status = StatusFinished
	if store.tasks["running"].Status != StatusRunning {
		t.Fatalf("ListRunningTasks returned original task instead of clone")
	}
}
