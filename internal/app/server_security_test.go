package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleTaskSubroutesRejectsLongPath(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/"+strings.Repeat("a", maxTaskSubroutePathLength+1), nil)
	rr := httptest.NewRecorder()
	s.handleTaskSubroutes(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for long task subroute path, got %d", rr.Code)
	}
}
