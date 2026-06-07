package app

import (
	"strings"
	"testing"
)

func TestUpdateSettingsRejectsOversizedModelTokens(t *testing.T) {
	store, err := NewStore(t.TempDir(), NewEventHub())
	if err != nil {
		t.Fatal(err)
	}
	settings := store.GetSettings()
	settings.MiniMaxToken = strings.Repeat("a", maxSettingsTokenBytes+1)
	if _, err := store.UpdateSettings(settings); err == nil {
		t.Fatal("UpdateSettings accepted oversized MiniMax token")
	}

	settings = store.GetSettings()
	settings.KimiToken = strings.Repeat("你", maxSettingsTokenBytes/3+1)
	if _, err := store.UpdateSettings(settings); err == nil {
		t.Fatal("UpdateSettings accepted oversized Kimi token")
	}
}
