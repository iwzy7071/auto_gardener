package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRejectCrossOriginAPIWrite(t *testing.T) {
	hub := NewEventHub()
	store, err := NewStore(t.TempDir(), hub)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, nil, t.TempDir(), hub)
	req := httptest.NewRequest(http.MethodPut, "http://gardener.local/api/settings", strings.NewReader(`{"logLevel":"quiet"}`))
	req.Header.Set("Origin", "https://evil.example")
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
}

func TestAllowSameOriginAPIWrite(t *testing.T) {
	hub := NewEventHub()
	store, err := NewStore(t.TempDir(), hub)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, nil, t.TempDir(), hub)
	req := httptest.NewRequest(http.MethodPut, "http://gardener.local/api/settings", strings.NewReader(`{"logLevel":"quiet"}`))
	req.Header.Set("Origin", "http://gardener.local")
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
}

func TestAllowDingTalkRobotWithoutBrowserOrigin(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "http://gardener.local/api/dingtalk/robot", strings.NewReader(`{}`))
	req.Header.Set("Origin", "https://oapi.dingtalk.com")
	rr := httptest.NewRecorder()
	rejectCrossOriginAPIWrites(next).ServeHTTP(rr, req)
	if !called || rr.Code != http.StatusNoContent {
		t.Fatalf("DingTalk route should bypass browser CSRF guard; called=%v status=%d", called, rr.Code)
	}
}
