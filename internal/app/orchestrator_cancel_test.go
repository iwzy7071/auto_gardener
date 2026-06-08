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

func TestTreeSummaryLimitsPromptContext(t *testing.T) {
	task := &Task{}
	longName := strings.Repeat("x", maxTreeSummaryFieldRunes+20)
	for i := 0; i < maxTreeSummaryEntries+5; i++ {
		task.Trees = append(task.Trees, &Tree{
			ID:        newID("tree"),
			Name:      longName,
			Forest:    i + 1,
			Status:    StatusFinished,
			FruitPath: strings.Repeat("/tmp/report", 30),
			Scope:     []string{strings.Repeat("scope", 80)},
		})
	}

	summary := treeSummary(task)
	if strings.Count(summary, "\n") != maxTreeSummaryEntries+1 {
		t.Fatalf("summary line count = %d, want %d", strings.Count(summary, "\n"), maxTreeSummaryEntries+1)
	}
	if !strings.Contains(summary, "已省略") {
		t.Fatalf("expected omitted marker, got %q", summary)
	}
	if strings.Contains(summary, longName) {
		t.Fatalf("long tree field was not truncated")
	}
}
