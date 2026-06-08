package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServeWorkspaceFileRejectsOversizedDownload(t *testing.T) {
	workspace := t.TempDir()
	path := filepath.Join(workspace, "huge.bin")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxWorkspaceDownloadBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/task/files?path=huge.bin&download=1", nil)
	rr := httptest.NewRecorder()
	server.serveWorkspaceFile(rr, req, &Task{WorkspacePath: workspace}, "huge.bin")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}
