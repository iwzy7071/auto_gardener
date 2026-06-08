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

func TestAllowReverseProxyHostWithoutPublicPort(t *testing.T) {
	hub := NewEventHub()
	store, err := NewStore(t.TempDir(), hub)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, nil, t.TempDir(), hub)
	req := httptest.NewRequest(http.MethodPut, "http://8.137.101.238/api/settings", strings.NewReader(`{"logLevel":"quiet"}`))
	req.Host = "8.137.101.238"
	req.Header.Set("Origin", "http://8.137.101.238:28081")
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("reverse-proxy same host with stripped port should be allowed; status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRejectSpoofedForwardedHostAPIWrite(t *testing.T) {
	hub := NewEventHub()
	store, err := NewStore(t.TempDir(), hub)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, nil, t.TempDir(), hub)
	req := httptest.NewRequest(http.MethodPut, "http://gardener.local/api/settings", strings.NewReader(`{"logLevel":"quiet"}`))
	req.Host = "gardener.local"
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("X-Forwarded-Host", "evil.example")
	req.Header.Set("X-Original-Host", "evil.example")
	req.Header.Set("Forwarded", `for=192.0.2.10;host="evil.example";proto=https`)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("spoofed forwarded host should be rejected; status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAllowConfiguredOriginAPIWrite(t *testing.T) {
	t.Setenv("AUTO_GARDENER_ALLOWED_ORIGINS", "http://8.137.101.238:28081")
	hub := NewEventHub()
	store, err := NewStore(t.TempDir(), hub)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, nil, t.TempDir(), hub)
	req := httptest.NewRequest(http.MethodPut, "http://127.0.0.1:8080/api/settings", strings.NewReader(`{"logLevel":"quiet"}`))
	req.Header.Set("Origin", "http://8.137.101.238:28081")
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("configured origin should be allowed; status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRejectSameHostDifferentExplicitPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://gardener.local:8080/api/tasks", nil)
	req.Host = "gardener.local:8080"
	req.Header.Set("Origin", "http://gardener.local:9999")
	if requestHasSameOrigin(req) {
		t.Fatal("same host with different explicit ports should not be considered same-origin")
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
