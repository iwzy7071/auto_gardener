package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestTaskDataPermissionsArePrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not reliable on Windows")
	}
	dataDir := t.TempDir()
	store, err := NewStore(dataDir, NewEventHub())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	taskID := "task_permissions"
	forestDir := filepath.Join(dataDir, "forests", taskID)
	task := &Task{
		ID:             taskID,
		Title:          "secret task",
		Prompt:         "contains user prompt",
		WorkspacePath:  filepath.Join(dataDir, "workspaces", taskID),
		ScratchPath:    filepath.Join(dataDir, "scratch", taskID),
		Status:         StatusRunning,
		GardenerStatus: StatusRunning,
		SchedulePath:   filepath.Join(forestDir, "gardener", "schedule.md"),
		LogPath:        filepath.Join(forestDir, "gardener", "log.md"),
		Messages:       []Message{{Role: RoleUser, Content: "private prompt"}},
		Trees: []*Tree{{
			ID:        "tree_permissions",
			TaskID:    taskID,
			Name:      "secret tree",
			Objective: "private objective",
			Status:    StatusRunning,
		}},
	}
	if err := store.AddTask(task); err != nil {
		t.Fatalf("AddTask: %v", err)
	}
	if err := store.WriteSchedule(taskID, "private schedule"); err != nil {
		t.Fatalf("WriteSchedule: %v", err)
	}
	store.AppendGardenerLog(taskID, "private log")
	store.AppendTreeProgress(taskID, "tree_permissions", "private progress")

	for _, path := range []string{
		filepath.Join(dataDir, "forests"),
		filepath.Join(dataDir, "scratch"),
		forestDir,
		filepath.Join(forestDir, "gardener"),
		filepath.Join(forestDir, "trees"),
		filepath.Join(forestDir, "trees", "tree_permissions"),
	} {
		assertPerm(t, path, privateDirMode)
	}
	for _, path := range []string{
		filepath.Join(forestDir, "forest.json"),
		filepath.Join(forestDir, "messages.json"),
		filepath.Join(forestDir, "gardener", "schedule.md"),
		filepath.Join(forestDir, "gardener", "log.md"),
		filepath.Join(forestDir, "trees", "tree_permissions", "tree.json"),
		filepath.Join(forestDir, "trees", "tree_permissions", "progress.log"),
	} {
		assertPerm(t, path, privateFileMode)
	}
}

func assertPerm(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %v, want %v", path, got, want)
	}
}

func TestEnsurePrivateDirRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on some Windows setups")
	}
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := ensurePrivateDir(link); err == nil {
		t.Fatal("ensurePrivateDir accepted symlink directory")
	}
}
