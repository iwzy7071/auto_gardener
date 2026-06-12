package app

import "testing"

func TestCompactTaskListExposesWorkspacePathOnly(t *testing.T) {
	tasks := compactTaskList([]*Task{{
		ID:            "task1",
		Prompt:        "secret prompt",
		WorkspacePath: "/Users/alice/project",
		ScratchPath:   "/tmp/GardenerScratch/task1",
		SchedulePath:  "/Users/alice/Desktop/forest_data/forests/task1/gardener/schedule.md",
		LogPath:       "/Users/alice/Desktop/forest_data/forests/task1/gardener/log.md",
		Messages:      []Message{{Content: "secret message"}},
		Trees: []*Tree{{
			ID:        "tree1",
			TaskID:    "task1",
			FruitPath: "/Users/alice/Desktop/forest_data/forests/task1/trees/tree1/fruit.md",
		}},
	}})
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	got := tasks[0]
	if got.WorkspacePath != "/Users/alice/project" {
		t.Fatalf("compact task should expose workspace path for UI, got %#v", got.WorkspacePath)
	}
	if got.Prompt != "" || got.ScratchPath != "" || got.SchedulePath != "" || got.LogPath != "" || got.Messages != nil {
		t.Fatalf("compact task exposed sensitive fields: %#v", got)
	}
	if len(got.Trees) != 1 || got.Trees[0].FruitPath != "ready" {
		t.Fatalf("compact tree should expose only fruit readiness, got %#v", got.Trees)
	}
}
