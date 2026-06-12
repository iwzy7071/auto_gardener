package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
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

func TestWorkspaceFileDiffShowsTrackedChangesOnly(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("old\nkeep\n"), 0600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "tracked.txt")
	runGit("commit", "-m", "initial")
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("new\nkeep\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "untracked.txt"), []byte("brand new\n"), 0600); err != nil {
		t.Fatal(err)
	}

	task := &Task{WorkspacePath: root, CreatedAt: time.Now().Add(-time.Minute)}
	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/task/files?path=tracked.txt&diff=1", nil)
	rr := httptest.NewRecorder()
	server.serveWorkspaceFile(rr, req, task, "tracked.txt")
	if rr.Code != http.StatusOK {
		t.Fatalf("diff expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	diff := rr.Body.String()
	if !strings.Contains(diff, "-old") || !strings.Contains(diff, "+new") {
		t.Fatalf("diff did not show tracked changes: %s", diff)
	}

	newReq := httptest.NewRequest(http.MethodGet, "/api/tasks/task/files?path=untracked.txt&diff=1", nil)
	newRR := httptest.NewRecorder()
	server.serveWorkspaceFile(newRR, newReq, task, "untracked.txt")
	if newRR.Code != http.StatusOK {
		t.Fatalf("new file diff expected 200, got %d: %s", newRR.Code, newRR.Body.String())
	}
	if strings.TrimSpace(newRR.Body.String()) != "" {
		t.Fatalf("new/untracked file should not render a modification diff: %s", newRR.Body.String())
	}
}
