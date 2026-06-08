package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"auto_gardener/internal/codex"
)

func TestCreateTaskRejectsWorkspaceOutsideAllowedRoots(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()
	t.Setenv("HOME", home)
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(store, codex.MockRunner{}, t.TempDir(), "")
	if _, err := orch.CreateTask("do work", outside); err == nil {
		t.Fatal("CreateTask accepted workspace outside allowed roots")
	}
}

func TestWorkspacePathAllowsHomePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	orch := NewOrchestrator(nil, codex.MockRunner{}, t.TempDir(), "")
	workspace := filepath.Join(home, "project")
	if !orch.isAllowedWorkspacePath(workspace) {
		t.Fatalf("home workspace was not allowed: %s", workspace)
	}
}

func TestWorkspacePathAllowsConfiguredRoot(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AUTO_GARDENER_ALLOWED_WORKSPACE_ROOTS", outside)
	orch := NewOrchestrator(nil, codex.MockRunner{}, t.TempDir(), "")
	workspace := filepath.Join(outside, "project")
	if !orch.isAllowedWorkspacePath(workspace) {
		t.Fatalf("configured workspace root was not allowed: %s", workspace)
	}
}

func TestServeWorkspaceFileRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on some Windows setups")
	}
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("SECRET"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked.txt")); err != nil {
		t.Fatal(err)
	}
	server := NewServer(nil, nil, t.TempDir(), nil)
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/task/files?path=linked.txt", nil)
	rr := httptest.NewRecorder()
	server.serveWorkspaceFile(rr, req, &Task{WorkspacePath: root}, "linked.txt")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
}
