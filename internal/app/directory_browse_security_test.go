package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDirectoryBrowseRejectsOutsideHome(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()
	t.Setenv("HOME", home)
	server := NewServer(nil, nil, t.TempDir(), nil)
	req := httptest.NewRequest(http.MethodGet, "/api/fs/dirs?path="+outside, nil)
	rr := httptest.NewRecorder()
	server.handleDirectoryBrowse(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
}

func TestDirectoryBrowseAllowsHomeChild(t *testing.T) {
	home := t.TempDir()
	child := filepath.Join(home, "projects")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	server := NewServer(nil, nil, t.TempDir(), nil)
	req := httptest.NewRequest(http.MethodGet, "/api/fs/dirs?path="+child, nil)
	rr := httptest.NewRecorder()
	server.handleDirectoryBrowse(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
}

func TestDirectoryBrowseRootOverride(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AUTO_GARDENER_ALLOW_DIRECTORY_BROWSE_ROOT", "1")
	server := NewServer(nil, nil, t.TempDir(), nil)
	req := httptest.NewRequest(http.MethodGet, "/api/fs/dirs?path="+outside, nil)
	rr := httptest.NewRecorder()
	server.handleDirectoryBrowse(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
}
