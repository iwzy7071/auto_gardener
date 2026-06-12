package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWorkspaceFileBrowserBlocksSecretFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("TOKEN=secret"), 0600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "result.txt"), []byte("ok"), 0600); err != nil {
		t.Fatalf("write result: %v", err)
	}
	task := &Task{WorkspacePath: root, CreatedAt: time.Now().Add(-time.Minute)}
	server := &Server{}

	listReq := httptest.NewRequest(http.MethodGet, "/api/tasks/task/files", nil)
	listRR := httptest.NewRecorder()
	server.listWorkspaceFiles(listRR, listReq, task)
	if listRR.Code != http.StatusOK {
		t.Fatalf("list expected 200, got %d: %s", listRR.Code, listRR.Body.String())
	}
	var listBody struct {
		Files []workspaceFileEntry `json:"files"`
	}
	if err := json.Unmarshal(listRR.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode file list: %v", err)
	}
	for _, file := range listBody.Files {
		if file.Path == ".env" {
			t.Fatalf("file list exposed .env: %s", listRR.Body.String())
		}
	}

	fileReq := httptest.NewRequest(http.MethodGet, "/api/tasks/task/files?path=.env", nil)
	fileRR := httptest.NewRecorder()
	server.serveWorkspaceFile(fileRR, fileReq, task, ".env")
	if fileRR.Code != http.StatusForbidden {
		t.Fatalf("secret file expected 403, got %d: %s", fileRR.Code, fileRR.Body.String())
	}
}

func TestWorkspaceFileBrowserSkipsPythonCacheFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg", "__pycache__"), 0755); err != nil {
		t.Fatalf("mkdir pycache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "__pycache__", "module.cpython-312.pyc"), []byte("cache"), 0600); err != nil {
		t.Fatalf("write pyc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "module.pyc"), []byte("cache"), 0600); err != nil {
		t.Fatalf("write top-level pyc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "result.txt"), []byte("ok"), 0600); err != nil {
		t.Fatalf("write result: %v", err)
	}
	task := &Task{WorkspacePath: root, CreatedAt: time.Now().Add(-time.Minute)}
	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/task/files", nil)
	rr := httptest.NewRecorder()
	server.listWorkspaceFiles(rr, req, task)
	if rr.Code != http.StatusOK {
		t.Fatalf("list expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var body struct {
		Files []workspaceFileEntry `json:"files"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode file list: %v", err)
	}
	var paths []string
	for _, file := range body.Files {
		paths = append(paths, file.Path)
		if strings.Contains(file.Path, "__pycache__") || strings.HasSuffix(file.Path, ".pyc") {
			t.Fatalf("file list exposed Python cache file %q: %s", file.Path, rr.Body.String())
		}
	}
	if len(paths) != 1 || paths[0] != "result.txt" {
		t.Fatalf("unexpected file list: %#v", paths)
	}
}
