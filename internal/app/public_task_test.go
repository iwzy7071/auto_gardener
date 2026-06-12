package app

import "testing"

func TestPublicTaskExposesWorkspacePathOnly(t *testing.T) {
	task := publicTask(&Task{
		ID:            "task1",
		Prompt:        "secret prompt",
		WorkspacePath: "/Users/alice/project",
		ScratchPath:   "/tmp/GardenerScratch/task1",
		SchedulePath:  "/Users/alice/Desktop/forest_data/forests/task1/gardener/schedule.md",
		LogPath:       "/Users/alice/Desktop/forest_data/forests/task1/gardener/log.md",
		Messages:      []Message{{Role: RoleUser, Content: "visible conversation"}},
		Trees: []*Tree{{
			ID:        "tree1",
			TaskID:    "task1",
			FruitPath: "/Users/alice/Desktop/forest_data/forests/task1/trees/tree1/fruit.md",
			GoalPath:  "/Users/alice/Desktop/forest_data/forests/task1/trees/tree1/goal.md",
		}},
	})
	if task.WorkspacePath != "/Users/alice/project" {
		t.Fatalf("public task should expose workspace path for UI, got %#v", task.WorkspacePath)
	}
	if task.Prompt != "" || task.ScratchPath != "" || task.SchedulePath != "" || task.LogPath != "" {
		t.Fatalf("public task exposed sensitive fields: %#v", task)
	}
	if len(task.Messages) != 1 {
		t.Fatalf("public task should keep messages for task detail UI")
	}
	if len(task.Trees) != 1 || task.Trees[0].FruitPath != "ready" || task.Trees[0].GoalPath != "" {
		t.Fatalf("public tree exposed paths or lost readiness: %#v", task.Trees)
	}
}
