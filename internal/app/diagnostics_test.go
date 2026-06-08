package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiagnosticsEndpointReturnsSafeSummary(t *testing.T) {
	t.Setenv("AUTO_GARDENER_CODEX_CMD", "definitely-missing-codex-secret-token-value")
	t.Setenv("AUTO_GARDENER_ALLOWED_ORIGINS", "https://garden.example")
	hub := NewEventHub()
	store, err := NewStore(t.TempDir(), hub)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, nil, t.TempDir(), hub)
	req := httptest.NewRequest(http.MethodGet, "/api/diagnostics", nil)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{`"items"`, `"summary"`, `"id":"data"`, `"id":"static"`, `"id":"codex"`, `"id":"power"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("diagnostics response missing %s: %s", want, body)
		}
	}
	if strings.Contains(body, "secret-token-value") {
		t.Fatalf("diagnostics leaked configured command value: %s", body)
	}
}
