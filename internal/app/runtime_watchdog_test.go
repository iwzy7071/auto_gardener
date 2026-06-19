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

func TestWatchdogStartsSilentGardenerReviewInsteadOfUserCue(t *testing.T) {
	t.Setenv("AUTO_GARDENER_WATCHDOG_STALE_SECONDS", "60")
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(store, codex.MockRunner{}, store.DataDir(), "")
	now := time.Now()
	old := now.Add(-2 * time.Minute)
	logDir := t.TempDir()
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
		LogPath:        logDir + "/log.md",
		SchedulePath:   logDir + "/schedule.md",
		Trees:          []*Tree{{ID: "tree", TaskID: "forest_watchdog", Status: StatusRunning, UpdatedAt: old}},
	}
	if err := store.AddTask(task); err != nil {
		t.Fatal(err)
	}
	orch.RunWatchdogOnce(now)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, ok := store.GetTask(task.ID)
		if ok && got.Status == StatusFinished {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	orch.RunWatchdogOnce(now.Add(10 * time.Second))
	got, ok := store.GetTask(task.ID)
	if !ok {
		t.Fatal("task missing")
	}
	if got.LastWatchdogAt == nil {
		t.Fatalf("LastWatchdogAt was not set")
	}
	for _, msg := range got.Messages {
		if msg.Role == RoleSystem && strings.Contains(msg.Content, "任务状态提示") {
			t.Fatalf("watchdog should not append user-facing stale cue; messages=%+v", got.Messages)
		}
	}
	count := 0
	for _, line := range got.GardenerProgress {
		if strings.Contains(line, "后台自查") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("watchdog self-review log count = %d, want 1; progress=%+v", count, got.GardenerProgress)
	}
}

func TestWatchdogDoesNotCancelActiveCLIProcess(t *testing.T) {
	t.Setenv("AUTO_GARDENER_WATCHDOG_STALE_SECONDS", "60")
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(store, codex.MockRunner{}, store.DataDir(), "")
	now := time.Now()
	old := now.Add(-2 * time.Minute)
	logDir := t.TempDir()
	task := &Task{
		ID:             "forest_watchdog_active_process",
		Title:          "watchdog active process",
		Prompt:         "watchdog active process",
		WorkspacePath:  t.TempDir(),
		ScratchPath:    t.TempDir(),
		Status:         StatusRunning,
		GardenerStatus: StatusRunning,
		CreatedAt:      old,
		UpdatedAt:      old,
		LastProgressAt: &old,
		LogPath:        logDir + "/log.md",
		SchedulePath:   logDir + "/schedule.md",
		Trees:          []*Tree{{ID: "tree", TaskID: "forest_watchdog_active_process", Status: StatusRunning, UpdatedAt: old}},
	}
	if err := store.AddTask(task); err != nil {
		t.Fatal(err)
	}
	cancelled := false
	orch.registerCancel(task.ID+":tree", func() { cancelled = true })
	defer orch.unregisterCancel(task.ID + ":tree")

	orch.RunWatchdogOnce(now)
	if cancelled {
		t.Fatal("watchdog canceled an active CLI process")
	}
	got, ok := store.GetTask(task.ID)
	if !ok {
		t.Fatal("task missing")
	}
	if got.Status != StatusRunning {
		t.Fatalf("watchdog should leave task running while process is active, got %s", got.Status)
	}
	if got.LastWatchdogAt == nil {
		t.Fatal("LastWatchdogAt was not set")
	}
	joined := strings.Join(got.GardenerProgress, "\n")
	if !strings.Contains(joined, "为避免主动中断") {
		t.Fatalf("expected non-interruption watchdog log, got progress=%+v", got.GardenerProgress)
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
	t.Setenv("AUTO_GARDENER_ALLOWED_WORKSPACE_ROOTS", workspace)
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
