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

func TestPublicSettingsReportsConfiguredTokensWithoutLeakingValues(t *testing.T) {
	settings := publicSettings(AppSettings{LogLevel: LogLevelQuiet, ModelMode: ModelModeMiniMax, MiniMaxToken: "minimax-secret", KimiToken: "kimi-secret"})
	if !settings.MiniMaxTokenConfigured || !settings.KimiTokenConfigured {
		t.Fatalf("configured flags = minimax %v kimi %v, want both true", settings.MiniMaxTokenConfigured, settings.KimiTokenConfigured)
	}
	if settings.MiniMaxToken != "" || settings.KimiToken != "" {
		t.Fatalf("public settings leaked token values: %#v", settings)
	}
}

func TestEnvProviderTokensSeedSettingsWhenMissing(t *testing.T) {
	t.Setenv("AUTO_GARDENER_MINIMAX_TOKEN", "minimax-env-secret")
	t.Setenv("AUTO_GARDENER_KIMI_TOKEN", "kimi-env-secret")
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	settings := store.GetSettings()
	if settings.MiniMaxToken != "minimax-env-secret" || settings.KimiToken != "kimi-env-secret" {
		t.Fatalf("settings tokens were not seeded from environment")
	}
	public := store.GetPublicSettings()
	if !public.MiniMaxTokenConfigured || !public.KimiTokenConfigured {
		t.Fatalf("public configured flags = minimax %v kimi %v, want both true", public.MiniMaxTokenConfigured, public.KimiTokenConfigured)
	}
	if public.MiniMaxToken != "" || public.KimiToken != "" {
		t.Fatalf("public settings leaked env tokens")
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
