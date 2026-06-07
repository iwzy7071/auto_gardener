package app

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"auto_gardener/internal/codex"
)

type errorRunner struct {
	output string
	err    error
}

func (r errorRunner) Run(ctx context.Context, req codex.RunRequest) codex.RunResult {
	return codex.RunResult{Output: r.output, Err: r.err}
}

func newTestTask(t *testing.T, store *Store, id string) *Task {
	t.Helper()
	forestDir := filepath.Join(store.DataDir(), "forests", id)
	task := &Task{
		ID:                 id,
		Title:              "test task",
		Prompt:             "test prompt",
		WorkspacePath:      t.TempDir(),
		ScratchPath:        t.TempDir(),
		CLIEngine:          CLIEngineCodex,
		ModelMode:          ModelModeDefault,
		Status:             StatusRunning,
		GardenerStatus:     StatusRunning,
		Forest:             0,
		MaxTreesPerForest:  6,
		MaxConcurrentTrees: 3,
		SchedulePath:       filepath.Join(forestDir, "gardener", "schedule.md"),
		LogPath:            filepath.Join(forestDir, "gardener", "log.md"),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	if err := store.AddTask(task); err != nil {
		t.Fatal(err)
	}
	return task
}

func TestCancelledGardenerPlanDoesNotAppendCLIFailureMessage(t *testing.T) {
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	newTestTask(t, store, "forest_cancelled_plan")
	orch := NewOrchestrator(store, errorRunner{err: errors.New("signal: killed")}, store.DataDir(), "")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	plan := orch.runGardenerPlan(ctx, "forest_cancelled_plan", "continue")
	if !plan.ForestFinished {
		t.Fatalf("cancelled plan should pause/finish current run")
	}
	got, ok := store.GetTask("forest_cancelled_plan")
	if !ok {
		t.Fatal("task missing")
	}
	for _, msg := range got.Messages {
		if strings.Contains(msg.Content, "底层 CLI 或模型连接失败") {
			t.Fatalf("cancelled obsolete run appended false CLI failure message: %+v", got.Messages)
		}
	}
	if len(got.GardenerProgress) == 0 || !strings.Contains(got.GardenerProgress[len(got.GardenerProgress)-1], "不向用户显示模型失败") {
		t.Fatalf("expected internal cancellation log, got progress=%+v", got.GardenerProgress)
	}
}

func TestActiveGardenerPlanErrorStillAppendsCLIFailureMessage(t *testing.T) {
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	newTestTask(t, store, "forest_active_error")
	orch := NewOrchestrator(store, errorRunner{err: errors.New("model unavailable")}, store.DataDir(), "")

	_ = orch.runGardenerPlan(context.Background(), "forest_active_error", "continue")
	got, ok := store.GetTask("forest_active_error")
	if !ok {
		t.Fatal("task missing")
	}
	for _, msg := range got.Messages {
		if strings.Contains(msg.Content, "底层 CLI 或模型连接失败") {
			return
		}
	}
	t.Fatalf("active runner error should still append actionable CLI failure message, got %+v", got.Messages)
}

type endlessPlanRunner struct{}

func (r endlessPlanRunner) Run(ctx context.Context, req codex.RunRequest) codex.RunResult {
	if req.Role == "gardener" {
		return codex.RunResult{Output: `{"message_to_user":"continue","forest_finished":false,"trees":[{"name":"Loop task","objective":"Keep going","prompt":"Keep going","scope":["loop"]}]}`}
	}
	return codex.RunResult{Output: "# Report\n\nGoal status: complete"}
}

func TestRunForestStopsAtAutomaticForestLimit(t *testing.T) {
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	newTestTask(t, store, "forest_loop_limit")
	orch := NewOrchestrator(store, endlessPlanRunner{}, store.DataDir(), "")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runID := "run_loop_limit"
	key := "forest_loop_limit:gardener:" + runID
	orch.registerRun("forest_loop_limit", runID, key, cancel)
	defer orch.unregisterCancel(key)

	orch.runForest(ctx, "forest_loop_limit", "continue", runID)

	got, ok := store.GetTask("forest_loop_limit")
	if !ok {
		t.Fatal("task missing")
	}
	if got.Forest != maxAutomaticForestsPerRun {
		t.Fatalf("forest count = %d, want %d", got.Forest, maxAutomaticForestsPerRun)
	}
	if got.Status != StatusFinished || got.GardenerStatus != StatusFinished {
		t.Fatalf("task should be paused/finished after loop limit, got status=%s gardener=%s", got.Status, got.GardenerStatus)
	}
	found := false
	for _, msg := range got.Messages {
		if msg.Role == RoleSystem && strings.Contains(msg.Content, "安全上限") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected safety limit system message, got %+v", got.Messages)
	}
}
