package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestUpdateSettingsEnforcesPrivateFileMode(t *testing.T) {
	dataDir := t.TempDir()
	store, err := NewStore(dataDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	settingsPath := store.settingsPath()
	if err := os.Chmod(settingsPath, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateSettings(AppSettings{LogLevel: LogLevelQuiet, ModelMode: ModelModeMiniMax, MiniMaxToken: "secret"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("settings mode = %o, want 600", got)
	}
}
