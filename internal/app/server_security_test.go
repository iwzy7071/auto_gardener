package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeWorkspaceFileRejectsLongPath(t *testing.T) {
	s := &Server{}
	task := &Task{WorkspacePath: t.TempDir()}
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/forest/files", nil)
	rr := httptest.NewRecorder()
	s.serveWorkspaceFile(rr, req, task, strings.Repeat("a", maxWorkspaceFilePathLength+1))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for long workspace file path, got %d", rr.Code)
	}
}
