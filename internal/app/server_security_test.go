package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeStaticAppRejectsLongPath(t *testing.T) {
	s := &Server{staticDir: t.TempDir()}
	req := httptest.NewRequest(http.MethodGet, "/"+strings.Repeat("a", maxStaticRequestPathLength+1), nil)
	rr := httptest.NewRecorder()
	s.serveStaticApp(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for long static path, got %d", rr.Code)
	}
}
