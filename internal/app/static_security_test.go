package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServeStaticAppRejectsPathOutsideStaticRoot(t *testing.T) {
	root := t.TempDir()
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("INDEX"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "secret.txt"), []byte("SECRET"), 0644); err != nil {
		t.Fatal(err)
	}
	server := NewServer(nil, nil, staticDir, nil)
	req := httptest.NewRequest(http.MethodGet, "/../secret.txt", nil)
	rr := httptest.NewRecorder()
	server.serveStaticApp(rr, req)
	if rr.Code == http.StatusOK || rr.Body.String() == "SECRET" {
		t.Fatalf("static handler served file outside static root: status=%d body=%q", rr.Code, rr.Body.String())
	}
}

func TestServeStaticAppRejectsDotfiles(t *testing.T) {
	staticDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("INDEX"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, ".env"), []byte("SECRET"), 0644); err != nil {
		t.Fatal(err)
	}
	server := NewServer(nil, nil, staticDir, nil)
	req := httptest.NewRequest(http.MethodGet, "/.env", nil)
	rr := httptest.NewRecorder()
	server.serveStaticApp(rr, req)
	if rr.Code == http.StatusOK || rr.Body.String() == "SECRET" {
		t.Fatalf("static handler served dotfile: status=%d body=%q", rr.Code, rr.Body.String())
	}
}
