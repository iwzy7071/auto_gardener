package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServeMarkdownRejectsNonMarkdownFiles(t *testing.T) {
	dataDir := t.TempDir()
	secretPath := filepath.Join(dataDir, "settings.json")
	if err := os.WriteFile(secretPath, []byte(`{"token":"secret"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	store := &Store{dataDir: dataDir}
	server := &Server{store: store}
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/task/gardener/log.md", nil)
	rec := httptest.NewRecorder()
	server.serveMarkdown(rec, req, secretPath)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected non-Markdown report path to be forbidden, got %d", rec.Code)
	}
}
