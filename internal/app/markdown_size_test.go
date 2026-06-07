package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServeMarkdownRejectsLargeReports(t *testing.T) {
	dataDir := t.TempDir()
	path := filepath.Join(dataDir, "forests", "forest_test", "gardener", "log.md")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxMarkdownReportSize + 1); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	store := &Store{tasks: make(map[string]*Task), dataDir: dataDir, events: NewEventHub(), settings: defaultSettings()}
	server := NewServer(store, nil, "", NewEventHub())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/forest_test/gardener/log.md", nil)

	server.serveMarkdown(rec, req, path)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "报告文件过大") {
		t.Fatalf("body = %s, want size error", rec.Body.String())
	}
}
