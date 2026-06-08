package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWorkspaceChangesAndDiff(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	workspace := t.TempDir()
	runGitForTest(t, workspace, "init")
	runGitForTest(t, workspace, "config", "user.email", "test@example.com")
	runGitForTest(t, workspace, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(workspace, "app.txt"), []byte("old\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitForTest(t, workspace, "add", "app.txt")
	runGitForTest(t, workspace, "commit", "-m", "initial")
	if err := os.WriteFile(filepath.Join(workspace, "app.txt"), []byte("old\nnew\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "notes.md"), []byte("notes\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".env"), []byte("TOKEN=secret\n"), 0600); err != nil {
		t.Fatal(err)
	}

	hub := NewEventHub()
	store, err := NewStore(t.TempDir(), hub)
	if err != nil {
		t.Fatal(err)
	}
	task := &Task{ID: "forest_git", Title: "git", WorkspacePath: workspace, Status: StatusFinished, GardenerStatus: StatusFinished, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := store.AddTask(task); err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, nil, t.TempDir(), hub)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/forest_git/changes", nil)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("changes status=%d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, `"path":"app.txt"`) || !strings.Contains(body, `"path":"notes.md"`) {
		t.Fatalf("changes should include tracked and untracked safe files: %s", body)
	}
	if strings.Contains(body, ".env") || strings.Contains(body, "secret") {
		t.Fatalf("changes leaked sensitive file: %s", body)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/tasks/forest_git/diff?path=app.txt", nil)
	rr = httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("diff status=%d body=%s", rr.Code, rr.Body.String())
	}
	body = rr.Body.String()
	if !strings.Contains(body, "+new") || strings.Contains(body, "TOKEN") {
		t.Fatalf("unexpected diff body: %s", body)
	}
}

func runGitForTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
