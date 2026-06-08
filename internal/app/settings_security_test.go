package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSettingsResponseRedactsTokens(t *testing.T) {
	hub := NewEventHub()
	store, err := NewStore(t.TempDir(), hub)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateSettings(AppSettings{LogLevel: LogLevelQuiet, ModelMode: ModelModeMiniMax, MiniMaxToken: "minimax-secret", KimiToken: "kimi-secret"}); err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, nil, t.TempDir(), hub)
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "minimax-secret") || strings.Contains(rr.Body.String(), "kimi-secret") {
		t.Fatalf("settings response leaked token: %s", rr.Body.String())
	}
}

func TestUpdateSettingsResponseRedactsTokens(t *testing.T) {
	hub := NewEventHub()
	store, err := NewStore(t.TempDir(), hub)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateSettings(AppSettings{LogLevel: LogLevelQuiet, ModelMode: ModelModeMiniMax, MiniMaxToken: "minimax-secret", KimiToken: "kimi-secret"}); err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, nil, t.TempDir(), hub)
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"logLevel":"quiet","modelMode":"default"}`))
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "minimax-secret") || strings.Contains(rr.Body.String(), "kimi-secret") {
		t.Fatalf("settings update response leaked token: %s", rr.Body.String())
	}
}

func TestUpdateSettingsPreservesTokensWhenOmitted(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateSettings(AppSettings{LogLevel: LogLevelQuiet, ModelMode: ModelModeMiniMax, MiniMaxToken: "minimax-secret", KimiToken: "kimi-secret"}); err != nil {
		t.Fatal(err)
	}
	updated, err := store.UpdateSettings(AppSettings{LogLevel: LogLevelDetailed, ModelMode: ModelModeDefault})
	if err != nil {
		t.Fatal(err)
	}
	if updated.MiniMaxToken != "minimax-secret" || updated.KimiToken != "kimi-secret" {
		b, _ := json.Marshal(updated)
		t.Fatalf("empty token update did not preserve tokens: %s", b)
	}
}
