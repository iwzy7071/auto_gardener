package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServeWorkspaceFileBlocksHiddenDownloads(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("TOKEN=secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	task := &Task{ID: "task", WorkspacePath: root}
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/task/files?path=.env&download=1", nil)
	rec := httptest.NewRecorder()
	(&Server{}).serveWorkspaceFile(rec, req, task, ".env")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected hidden download to be forbidden, got %d", rec.Code)
	}
}
