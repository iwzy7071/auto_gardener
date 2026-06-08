package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

func TestServeStaticAppRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on some Windows setups")
	}
	root := t.TempDir()
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(outside, []byte("SECRET"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(staticDir, "linked.txt")); err != nil {
		t.Fatal(err)
	}
	server := NewServer(nil, nil, staticDir, nil)
	req := httptest.NewRequest(http.MethodGet, "/linked.txt", nil)
	rr := httptest.NewRecorder()
	server.serveStaticApp(rr, req)
	if rr.Code == http.StatusOK || rr.Body.String() == "SECRET" {
		t.Fatalf("static handler served symlink escape: status=%d body=%q", rr.Code, rr.Body.String())
	}
}
