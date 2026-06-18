package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"auto_gardener/internal/codex"
)

func TestCreateTaskUsesUniqueWorkspaceUnderRequestedSaveLocation(t *testing.T) {
	base := t.TempDir()
	t.Setenv("AUTO_GARDENER_ALLOWED_WORKSPACE_ROOTS", base)
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(store, codex.MockRunner{}, store.DataDir(), "")

	taskA, err := orch.CreateTask("帮我看看", base)
	if err != nil {
		t.Fatal(err)
	}
	taskB, err := orch.CreateTask("处理一下", base)
	if err != nil {
		t.Fatal(err)
	}
	if taskA.WorkspacePath == base || taskB.WorkspacePath == base {
		t.Fatalf("tasks should not use shared base as workspace: a=%s b=%s base=%s", taskA.WorkspacePath, taskB.WorkspacePath, base)
	}
	if taskA.WorkspacePath == taskB.WorkspacePath {
		t.Fatalf("tasks share workspace: %s", taskA.WorkspacePath)
	}
	for _, task := range []*Task{taskA, taskB} {
		if filepath.Dir(task.WorkspacePath) != base {
			t.Fatalf("workspace %s should be a direct child of %s", task.WorkspacePath, base)
		}
		if filepath.Base(task.WorkspacePath) != task.ID {
			t.Fatalf("workspace %s should be named by task id %s", task.WorkspacePath, task.ID)
		}
	}
}

func TestDefaultDesktopOutputPathIsTaskSubdirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	desktop := filepath.Join(home, "Desktop")

	got := defaultOutputPathForPrompt("保存到桌面", "forest_test", "title")
	if got == desktop {
		t.Fatalf("desktop output should use an isolated task subdirectory, got desktop root")
	}
	if filepath.Dir(got) != desktop || filepath.Base(got) != "forest_test" {
		t.Fatalf("unexpected desktop output path: %s", got)
	}
}

func TestWorkspaceFileBrowserHidesOtherTaskWorkspaceSubdirectories(t *testing.T) {
	dataDir := t.TempDir()
	store, err := NewStore(dataDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "own.txt"), []byte("own"), 0600); err != nil {
		t.Fatal(err)
	}
	otherRoot := filepath.Join(root, "forest_other")
	if err := os.MkdirAll(otherRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherRoot, "secret.txt"), []byte("other"), 0600); err != nil {
		t.Fatal(err)
	}
	taskA := &Task{ID: "forest_current", WorkspacePath: root, ScratchPath: filepath.Join(t.TempDir(), "scratch-a")}
	taskB := &Task{ID: "forest_other", WorkspacePath: otherRoot, ScratchPath: filepath.Join(t.TempDir(), "scratch-b")}
	if err := store.AddTask(taskA); err != nil {
		t.Fatal(err)
	}
	if err := store.AddTask(taskB); err != nil {
		t.Fatal(err)
	}
	server := &Server{store: store}

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/forest_current/files", nil)
	rr := httptest.NewRecorder()
	server.listWorkspaceFiles(rr, req, taskA)
	if rr.Code != http.StatusOK {
		t.Fatalf("list expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var body struct {
		Files []workspaceFileEntry `json:"files"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	paths := make(map[string]bool)
	for _, file := range body.Files {
		paths[file.Path] = true
	}
	if !paths["own.txt"] {
		t.Fatalf("own file missing from list: %s", rr.Body.String())
	}
	if paths["forest_other/secret.txt"] {
		t.Fatalf("other task workspace leaked into file list: %s", rr.Body.String())
	}

	fileReq := httptest.NewRequest(http.MethodGet, "/api/tasks/forest_current/files?path=forest_other/secret.txt", nil)
	fileRR := httptest.NewRecorder()
	server.serveWorkspaceFile(fileRR, fileReq, taskA, "forest_other/secret.txt")
	if fileRR.Code != http.StatusForbidden {
		t.Fatalf("other task file should be forbidden, got %d: %s", fileRR.Code, fileRR.Body.String())
	}
}
