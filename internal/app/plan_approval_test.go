package app

import (
	"testing"
	"time"

	"auto_gardener/internal/codex"
)

func waitForTaskState(t *testing.T, store *Store, taskID string, ok func(*Task) bool) *Task {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		task, found := store.GetTask(taskID)
		if found && ok(task) {
			return task
		}
		time.Sleep(20 * time.Millisecond)
	}
	task, _ := store.GetTask(taskID)
	t.Fatalf("task %s did not reach expected state; last=%#v", taskID, task)
	return nil
}

func TestCreateTaskPlanOnlyWaitsForApproval(t *testing.T) {
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	t.Setenv("AUTO_GARDENER_ALLOWED_WORKSPACE_ROOTS", workspace)
	orch := NewOrchestrator(store, codex.MockRunner{}, store.DataDir(), "")

	created, err := orch.CreateTaskWithOptions("create a mock deliverable", workspace, CreateTaskOptions{PlanOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	got := waitForTaskState(t, store, created.ID, func(task *Task) bool {
		return task.Status == StatusPendingApproval
	})
	if got.GardenerStatus != StatusFinished {
		t.Fatalf("gardener status = %q, want Finished", got.GardenerStatus)
	}
	if len(got.PendingPlan) != 1 {
		t.Fatalf("pending plan length = %d, want 1", len(got.PendingPlan))
	}
	if len(got.Trees) != 0 {
		t.Fatalf("plan-only task should not run trees before approval, got %d", len(got.Trees))
	}
}

func TestApproveTaskPlanRunsPendingPlan(t *testing.T) {
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	t.Setenv("AUTO_GARDENER_ALLOWED_WORKSPACE_ROOTS", workspace)
	orch := NewOrchestrator(store, codex.MockRunner{}, store.DataDir(), "")
	created, err := orch.CreateTaskWithOptions("create a mock deliverable", workspace, CreateTaskOptions{PlanOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	waitForTaskState(t, store, created.ID, func(task *Task) bool {
		return task.Status == StatusPendingApproval && len(task.PendingPlan) > 0
	})

	approved, err := orch.ApproveTaskPlan(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != StatusRunning || len(approved.PendingPlan) != 0 {
		t.Fatalf("approved task = status %q pending %d", approved.Status, len(approved.PendingPlan))
	}
	finished := waitForTaskState(t, store, created.ID, func(task *Task) bool {
		return task.Status == StatusFinished && len(task.Trees) >= 2
	})
	if len(finished.PendingPlan) != 0 {
		t.Fatalf("pending plan should be cleared, got %d", len(finished.PendingPlan))
	}
}
