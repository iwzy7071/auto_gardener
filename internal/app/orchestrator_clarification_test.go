package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"auto_gardener/internal/codex"
)

type clarificationRunner struct{}

func (r clarificationRunner) Run(ctx context.Context, req codex.RunRequest) codex.RunResult {
	if strings.Contains(req.Prompt, "Git 统一初始化") {
		return codex.RunResult{Output: "git init skipped"}
	}
	if req.Role == "gardener" {
		return codex.RunResult{Output: `{"message_to_user":"","forest_finished":true,"needs_clarification":true,"clarification_question":"请确认要改哪个页面，以及期望达到什么效果？","trees":[]}`}
	}
	return codex.RunResult{Output: "# report\n\nGoal status: complete\n"}
}

func TestCreateTaskPausesForObviouslyVaguePrompt(t *testing.T) {
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(store, codex.MockRunner{}, store.DataDir(), "")
	workspace := t.TempDir()
	t.Setenv("AUTO_GARDENER_ALLOWED_WORKSPACE_ROOTS", workspace)

	task, err := orch.CreateTask("帮我看看", workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !task.AwaitingUserInput {
		t.Fatalf("task should wait for user input: %+v", task)
	}
	if task.Status != StatusFinished || task.GardenerStatus != StatusFinished {
		t.Fatalf("awaiting task should be paused as Finished, got task=%s gardener=%s", task.Status, task.GardenerStatus)
	}
	if len(task.Trees) != 0 {
		t.Fatalf("vague prompt should not dispatch subtasks, got %d", len(task.Trees))
	}
	if task.Runtime == nil || task.Runtime.Phase != "awaiting_user" || task.Runtime.CanResume {
		t.Fatalf("unexpected awaiting runtime: %+v", task.Runtime)
	}
	if !lastGardenerMessageContains(task, "补充") {
		t.Fatalf("expected clarification message, got %+v", task.Messages)
	}
}

func TestGardenerPlanCanAskClarificationDuringExecution(t *testing.T) {
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(store, clarificationRunner{}, store.DataDir(), "")
	workspace := t.TempDir()
	t.Setenv("AUTO_GARDENER_ALLOWED_WORKSPACE_ROOTS", workspace)

	task, err := orch.CreateTask("请根据当前项目做下一步产品优化", workspace)
	if err != nil {
		t.Fatal(err)
	}
	got := waitForTask(t, store, task.ID, func(t *Task) bool {
		return t.AwaitingUserInput
	})
	if got.Status != StatusFinished || len(got.Trees) != 0 {
		t.Fatalf("clarification should pause without subtasks, status=%s trees=%d", got.Status, len(got.Trees))
	}
	if !lastGardenerMessageContains(got, "请确认要改哪个页面") {
		t.Fatalf("expected planner clarification, got %+v", got.Messages)
	}
}

func TestReplyToClarificationClearsAwaitingAndContinues(t *testing.T) {
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(store, codex.MockRunner{}, store.DataDir(), "")
	workspace := t.TempDir()
	t.Setenv("AUTO_GARDENER_ALLOWED_WORKSPACE_ROOTS", workspace)

	task, err := orch.CreateTask("处理一下", workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !task.AwaitingUserInput {
		t.Fatalf("expected task to await clarification: %+v", task)
	}
	if _, err := orch.ResumeTask(task.ID); err == nil {
		t.Fatal("ResumeTask should reject awaiting clarification tasks")
	}
	if _, err := orch.SendMessage(task.ID, "请创建一个 mock deliverable，并完成验证。"); err != nil {
		t.Fatal(err)
	}
	got := waitForTask(t, store, task.ID, func(t *Task) bool {
		return t.Status == StatusFinished && !t.AwaitingUserInput && len(t.Trees) >= 2
	})
	if got.AwaitingUserInput {
		t.Fatalf("reply should clear awaiting flag: %+v", got)
	}
	if got.Runtime == nil || !got.Runtime.CanResume {
		t.Fatalf("finished continued task should allow resume: %+v", got.Runtime)
	}
}

func TestProgressQueryWhileAwaitingClarificationDoesNotStartRun(t *testing.T) {
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(store, codex.MockRunner{}, store.DataDir(), "")
	workspace := t.TempDir()
	t.Setenv("AUTO_GARDENER_ALLOWED_WORKSPACE_ROOTS", workspace)

	task, err := orch.CreateTask("改一下", workspace)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := orch.SendMessage(task.ID, "现在进度如何？"); err != nil {
		t.Fatal(err)
	}
	got, ok := store.GetTask(task.ID)
	if !ok {
		t.Fatal("task missing")
	}
	if !got.AwaitingUserInput || got.Status != StatusFinished {
		t.Fatalf("progress query should keep task awaiting, got status=%s awaiting=%v", got.Status, got.AwaitingUserInput)
	}
	if len(got.Trees) != 0 {
		t.Fatalf("progress query should not dispatch subtasks, got %d", len(got.Trees))
	}
	if !lastGardenerMessageContains(got, "等你补充需求") {
		t.Fatalf("expected awaiting progress answer, got %+v", got.Messages)
	}
}

func waitForTask(t *testing.T, store *Store, taskID string, done func(*Task) bool) *Task {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var last *Task
	for time.Now().Before(deadline) {
		got, ok := store.GetTask(taskID)
		if !ok {
			t.Fatalf("task %s missing", taskID)
		}
		last = got
		if done(got) {
			return got
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for task; last=%+v", last)
	return last
}

func lastGardenerMessageContains(task *Task, needle string) bool {
	for i := len(task.Messages) - 1; i >= 0; i-- {
		if task.Messages[i].Role == RoleGardener {
			return strings.Contains(task.Messages[i].Content, needle)
		}
	}
	return false
}
