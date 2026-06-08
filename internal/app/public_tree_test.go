package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPublicTreeRedactsInternalPaths(t *testing.T) {
	tree := publicTree(&Tree{
		ID:        "tree1",
		TaskID:    "task1",
		FruitPath: "/Users/alice/Desktop/forest_data/forests/task1/trees/tree1/fruit.md",
		GoalPath:  "/Users/alice/Desktop/forest_data/forests/task1/trees/tree1/goal.md",
	})
	if tree.FruitPath != "ready" || tree.GoalPath != "" {
		t.Fatalf("public tree exposed paths: %#v", tree)
	}
}

func TestTreeEndpointRedactsInternalPaths(t *testing.T) {
	hub := NewEventHub()
	store, err := NewStore(t.TempDir(), hub)
	if err != nil {
		t.Fatal(err)
	}
	pathRoot := t.TempDir()
	task := &Task{
		ID:             "task1",
		Title:          "task",
		Status:         StatusRunning,
		GardenerStatus: StatusRunning,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Trees: []*Tree{{
			ID:        "tree1",
			TaskID:    "task1",
			Status:    StatusFinished,
			FruitPath: filepath.Join(pathRoot, "fruit.md"),
			GoalPath:  filepath.Join(pathRoot, "goal.md"),
			UpdatedAt: time.Now(),
		}},
	}
	if err := store.AddTask(task); err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, nil, t.TempDir(), hub)
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/task1/trees/tree1", nil)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Tree Tree `json:"tree"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Tree.FruitPath != "ready" || body.Tree.GoalPath != "" {
		t.Fatalf("tree endpoint exposed paths: %#v", body.Tree)
	}
	if strings.Contains(rr.Body.String(), pathRoot) {
		t.Fatalf("tree endpoint response leaked internal path: %s", rr.Body.String())
	}
}
