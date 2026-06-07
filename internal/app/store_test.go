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
