package app

import (
	"path/filepath"
	"strings"
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

func TestCreateTaskRejectsOversizedWorkspacePath(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(store, codex.MockRunner{}, t.TempDir(), "")
	tooLong := strings.Repeat("a", maxWorkspacePathBytes+1)
	if _, err := orch.CreateTask("do work", tooLong); err == nil {
		t.Fatal("CreateTask accepted oversized workspace path")
	}
}
