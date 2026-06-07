package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthDoesNotExposeDataDir(t *testing.T) {
	store, err := NewStore(t.TempDir(), NewEventHub())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	server := NewServer(store, nil, "", NewEventHub())
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()

	server.handleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if _, ok := body["dataDir"]; ok {
		t.Fatalf("health response exposes dataDir: %s", rr.Body.String())
	}
}
