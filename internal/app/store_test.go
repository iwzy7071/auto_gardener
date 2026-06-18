package app

import (
	"os"
	"testing"
)

func TestSettingsFileUsesOwnerOnlyPermissions(t *testing.T) {
	store, err := NewStore(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateSettings(AppSettings{LogLevel: LogLevelQuiet, ModelMode: ModelModeMiniMax, MiniMaxToken: "secret"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(store.settingsPath())
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("settings permissions = %o, want 600", got)
	}
}

func TestNormalizeModelModeMigratesLegacyMiniMax(t *testing.T) {
	for _, input := range []ModelMode{"minimaxm2.7", "minimax-m2.7", "minimaxm3", "minimax-m3", ModelModeMiniMax} {
		if got := normalizeModelMode(input); got != ModelModeMiniMax {
			t.Fatalf("normalizeModelMode(%q) = %q, want %q", input, got, ModelModeMiniMax)
		}
	}
}

func TestNormalizeModelModeAcceptsKimiAliases(t *testing.T) {
	for _, input := range []ModelMode{"kimi-k2.7", "kimi-k2.7-code", "kimik2.7", "kimik2.7-code", "kimik2.6", "kimi-k2.6", "kimi-coding", ModelModeKimi} {
		if got := normalizeModelMode(input); got != ModelModeKimi {
			t.Fatalf("normalizeModelMode(%q) = %q, want %q", input, got, ModelModeKimi)
		}
	}
}
