package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIResponsesAreNoStore(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()
	noStoreAPIResponses(next).ServeHTTP(rr, req)
	if !called {
		t.Fatal("next handler was not called")
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if got := rr.Header().Get("Pragma"); got != "no-cache" {
		t.Fatalf("Pragma = %q, want no-cache", got)
	}
}

func TestNonAPIResponsesDoNotGetNoStore(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	noStoreAPIResponses(next).ServeHTTP(rr, req)
	if got := rr.Header().Get("Cache-Control"); got != "" {
		t.Fatalf("Cache-Control = %q, want empty", got)
	}
}
