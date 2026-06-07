package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPITokenRequiredWhenConfigured(t *testing.T) {
	t.Setenv("AUTO_GARDENER_AUTH_TOKEN", "secret-token")
	hub := NewEventHub()
	store, err := NewStore(t.TempDir(), hub)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, nil, t.TempDir(), hub)
	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
}

func TestAPITokenAllowsBearerHeader(t *testing.T) {
	t.Setenv("AUTO_GARDENER_AUTH_TOKEN", "secret-token")
	hub := NewEventHub()
	store, err := NewStore(t.TempDir(), hub)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, nil, t.TempDir(), hub)
	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
}

func TestAPITokenAllowsQueryForEventSource(t *testing.T) {
	t.Setenv("AUTO_GARDENER_AUTH_TOKEN", "secret-token")
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/task_1/events?authToken=secret-token", nil)
	if !requestHasAPIToken(req, "secret-token") {
		t.Fatal("query authToken was not accepted")
	}
}

func TestAPITokenSkipsHealthAndDingTalk(t *testing.T) {
	t.Setenv("AUTO_GARDENER_AUTH_TOKEN", "secret-token")
	called := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called++; w.WriteHeader(http.StatusNoContent) })
	for _, path := range []string{"/api/health", "/api/dingtalk/robot"} {
		req := httptest.NewRequest(http.MethodGet, path, strings.NewReader(`{}`))
		rr := httptest.NewRecorder()
		requireAPIToken(next).ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("%s status = %d, want %d", path, rr.Code, http.StatusNoContent)
		}
	}
	if called != 2 {
		t.Fatalf("next called %d times, want 2", called)
	}
}
