package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadJSONRejectsOversizedStateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.json")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxJSONStateFileBytes + 1); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	var out any
	err = readJSON(path, &out)
	if err == nil || !strings.Contains(err.Error(), "JSON state file too large") {
		t.Fatalf("readJSON error = %v, want oversized state error", err)
	}
}

func TestReadJSONAcceptsSmallStateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"logLevel":"quiet"}`), 0600); err != nil {
		t.Fatal(err)
	}
	var settings AppSettings
	if err := readJSON(path, &settings); err != nil {
		t.Fatal(err)
	}
	if settings.LogLevel != LogLevelQuiet {
		t.Fatalf("settings = %+v", settings)
	}
}
