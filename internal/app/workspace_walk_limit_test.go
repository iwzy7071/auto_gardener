package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListWorkspaceFilesLimitsWalkEntries(t *testing.T) {
	workspace := t.TempDir()
	for i := 0; i < maxWorkspaceWalkEntries+10; i++ {
		name := filepath.Join(workspace, fmt.Sprintf("a%05d.tmp", i))
		if err := os.WriteFile(name, []byte("noise"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(workspace, "z-visible.txt"), []byte("visible"), 0600); err != nil {
		t.Fatal(err)
	}

	store := &Store{tasks: make(map[string]*Task), dataDir: t.TempDir(), events: NewEventHub(), settings: defaultSettings()}
	server := NewServer(store, nil, "", NewEventHub())
	task := &Task{ID: "forest_test", WorkspacePath: workspace, CreatedAt: time.Now()}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/forest_test/files", nil)

	server.listWorkspaceFiles(rec, req, task)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body struct {
		Files []workspaceFileEntry `json:"files"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, file := range body.Files {
		if file.Path == "z-visible.txt" {
			t.Fatalf("walk continued past entry limit and returned %q", file.Path)
		}
	}
}
