package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSettingsFileUsesOwnerOnlyPermissions(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateSettings(AppSettings{LogLevel: LogLevelQuiet, ModelMode: ModelModeMiniMax, MiniMaxToken: "secret"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(store.settingsPath())
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("settings permissions = %o, want 600", got)
	}
}

func TestNormalizeModelModeMigratesLegacyMiniMax(t *testing.T) {
	for _, input := range []ModelMode{"minimaxm2.7", "minimax-m2.7", "minimaxm3", "minimax-m3", ModelModeMiniMax} {
		if got := normalizeModelMode(input); got != ModelModeMiniMax {
			t.Fatalf("normalizeModelMode(%q) = %q, want %q", input, got, ModelModeMiniMax)
		}
	}
}

func TestNormalizeModelModeAcceptsKimiAliases(t *testing.T) {
	for _, input := range []ModelMode{"kimi-k2.7", "kimi-k2.7-code", "kimik2.7", "kimik2.7-code", "kimik2.6", "kimi-k2.6", "kimi-coding", ModelModeKimi} {
		if got := normalizeModelMode(input); got != ModelModeKimi {
			t.Fatalf("normalizeModelMode(%q) = %q, want %q", input, got, ModelModeKimi)
		}
	}
}

func TestUpdateSettingsAppliesRuntimeSelectionToExistingTasks(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	forestDir := filepath.Join(store.DataDir(), "forests", "forest_existing")
	task := &Task{
		ID:                 "forest_existing",
		Title:              "existing task",
		Prompt:             "do work",
		WorkspacePath:      t.TempDir(),
		ScratchPath:        t.TempDir(),
		CLIEngine:          CLIEngineClaude,
		ModelMode:          ModelModeKimi,
		Status:             StatusFinished,
		GardenerStatus:     StatusFinished,
		Forest:             1,
		MaxTreesPerForest:  5,
		MaxConcurrentTrees: 3,
		SchedulePath:       filepath.Join(forestDir, "gardener", "schedule.md"),
		LogPath:            filepath.Join(forestDir, "gardener", "log.md"),
		CreatedAt:          time.Now().Add(-time.Hour),
		UpdatedAt:          time.Now().Add(-time.Hour),
	}
	if err := store.AddTask(task); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateSettings(AppSettings{LogLevel: LogLevelQuiet, ModelMode: ModelModeMiniMax, CLIEngine: CLIEngineCodex, MiniMaxToken: "secret"}); err != nil {
		t.Fatal(err)
	}
	got, ok := store.GetTask(task.ID)
	if !ok {
		t.Fatal("task missing")
	}
	if got.ModelMode != ModelModeMiniMax || got.CLIEngine != CLIEngineClaude {
		t.Fatalf("task runtime selection = cli %q model %q, want claude / MiniMax-M3", got.CLIEngine, got.ModelMode)
	}
	var disk Task
	if err := readJSON(filepath.Join(forestDir, "forest.json"), &disk); err != nil {
		t.Fatal(err)
	}
	if disk.ModelMode != ModelModeMiniMax || disk.CLIEngine != CLIEngineClaude {
		b, _ := json.Marshal(disk)
		t.Fatalf("persisted task runtime selection = %s", b)
	}
}
