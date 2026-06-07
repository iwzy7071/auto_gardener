package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServeTaskMarkdownRejectsOtherTaskReports(t *testing.T) {
	dataDir := t.TempDir()
	otherDir := filepath.Join(dataDir, "forests", "forest_other", "gardener")
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatal(err)
	}
	otherReport := filepath.Join(otherDir, "log.md")
	if err := os.WriteFile(otherReport, []byte("# other task"), 0o644); err != nil {
		t.Fatal(err)
	}

	server := &Server{store: &Store{dataDir: dataDir}}
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/forest_current/gardener/log.md", nil)
	rec := httptest.NewRecorder()
	server.serveTaskMarkdown(rec, req, "forest_current", otherReport)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected cross-task report path to be forbidden, got %d", rec.Code)
	}
}
