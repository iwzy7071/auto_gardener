package app

import (
	"strings"
	"testing"
	"time"

	"auto_gardener/internal/codex"
)

func TestBuildTaskRuntimeDetectsStaleRunningTask(t *testing.T) {
	now := time.Now()
	old := now.Add(-12 * time.Minute)
	task := &Task{
		ID:             "forest_test",
		Status:         StatusRunning,
		GardenerStatus: StatusRunning,
		CreatedAt:      now.Add(-30 * time.Minute),
		UpdatedAt:      old,
		LastProgressAt: &old,
		Trees: []*Tree{{
			ID:        "tree_test",
			Status:    StatusRunning,
			UpdatedAt: old,
		}},
	}
	rt := buildTaskRuntime(task, now)
	if rt == nil {
		t.Fatal("runtime is nil")
	}
	if rt.Severity != runtimeSeverityWarning {
		t.Fatalf("severity = %q, want warning; cue=%s", rt.Severity, rt.Cue)
	}
	if rt.RunningTrees != 1 || rt.TotalTrees != 1 {
		t.Fatalf("tree counts = running %d total %d", rt.RunningTrees, rt.TotalTrees)
	}
	if !rt.CanAskProgress || rt.CanResume {
		t.Fatalf("unexpected actions: ask=%v resume=%v", rt.CanAskProgress, rt.CanResume)
	}
}

func TestWatchdogAddsUserCueOncePerIdlePeriod(t *testing.T) {
	t.Setenv("AUTO_GARDENER_WATCHDOG_STALE_SECONDS", "1")
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(store, codex.MockRunner{}, store.DataDir(), "")
	now := time.Now()
	old := now.Add(-2 * time.Minute)
	task := &Task{
		ID:             "forest_watchdog",
		Title:          "watchdog",
		Prompt:         "watchdog",
		WorkspacePath:  t.TempDir(),
		ScratchPath:    t.TempDir(),
		Status:         StatusRunning,
		GardenerStatus: StatusRunning,
		CreatedAt:      old,
		UpdatedAt:      old,
		LastProgressAt: &old,
		LogPath:        t.TempDir() + "/log.md",
		SchedulePath:   t.TempDir() + "/schedule.md",
		Trees:          []*Tree{{ID: "tree", TaskID: "forest_watchdog", Status: StatusRunning, UpdatedAt: old}},
	}
	if err := store.AddTask(task); err != nil {
		t.Fatal(err)
	}
	orch.RunWatchdogOnce(now)
	orch.RunWatchdogOnce(now.Add(10 * time.Second))
	got, ok := store.GetTask(task.ID)
	if !ok {
		t.Fatal("task missing")
	}
	count := 0
	for _, msg := range got.Messages {
		if msg.Role == RoleSystem && strings.Contains(msg.Content, "任务状态提示") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("watchdog cue count = %d, want 1; messages=%+v", count, got.Messages)
	}
}

func TestMockRunnerCompletesEndToEndTask(t *testing.T) {
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(store, codex.MockRunner{}, store.DataDir(), "")
	workspace := t.TempDir()
	task, err := orch.CreateTask("Create a mock deliverable", workspace)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		got, ok := store.GetTask(task.ID)
		if ok && got.Status == StatusFinished {
			if len(got.Trees) < 2 {
				t.Fatalf("expected implementation + validation trees, got %d", len(got.Trees))
			}
			if got.Runtime == nil || !got.Runtime.CanResume {
				t.Fatalf("finished task runtime should allow resume: %+v", got.Runtime)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	got, _ := store.GetTask(task.ID)
	t.Fatalf("task did not finish; status=%v progress=%v", got.Status, got.GardenerProgress)
}

func TestGetenvDurationSecondsRejectsOverLimit(t *testing.T) {
	const key = "AUTO_GARDENER_TEST_WATCHDOG_SECONDS"
	fallback := 42 * time.Second
	t.Setenv(key, "604801")
	if got := getenvDurationSeconds(key, fallback); got != fallback {
		t.Fatalf("getenvDurationSeconds over-limit = %v, want fallback %v", got, fallback)
	}
	t.Setenv(key, "604800")
	if got := getenvDurationSeconds(key, fallback); got != 604800*time.Second {
		t.Fatalf("getenvDurationSeconds max = %v, want 604800s", got)
	}
	t.Setenv(key, "999999999999999999999999")
	if got := getenvDurationSeconds(key, fallback); got != fallback {
		t.Fatalf("getenvDurationSeconds huge = %v, want fallback %v", got, fallback)
	}
}
