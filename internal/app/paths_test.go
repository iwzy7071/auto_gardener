package app

import (
	"path/filepath"
	"testing"
)

func TestSafeConfiguredDataDirRejectsUnsafePaths(t *testing.T) {
	for _, path := range []string{"", ".", "..", "../forest_data", "tmp/../forest_data", string(filepath.Separator)} {
		if got, ok := safeConfiguredDataDir(path); ok {
			t.Fatalf("safeConfiguredDataDir(%q) = %q, true; want rejected", path, got)
		}
	}
}

func TestSafeConfiguredDataDirAcceptsNormalPath(t *testing.T) {
	want, err := filepath.Abs(filepath.Join("tmp", "forest_data"))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := safeConfiguredDataDir(filepath.Join("tmp", "forest_data"))
	if !ok {
		t.Fatal("expected normal data dir to be accepted")
	}
	if got != want {
		t.Fatalf("safeConfiguredDataDir returned %q, want %q", got, want)
	}
}
