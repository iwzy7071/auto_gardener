package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeConfiguredStaticDirRejectsUnsafePaths(t *testing.T) {
	for _, path := range []string{"", ".", "..", "../web/static", "tmp/../web/static", string(os.PathSeparator)} {
		if got, ok := safeConfiguredStaticDir(path); ok {
			t.Fatalf("safeConfiguredStaticDir(%q) = %q, true; want rejected", path, got)
		}
	}
}

func TestSafeConfiguredStaticDirRequiresIndex(t *testing.T) {
	dir := t.TempDir()
	if got, ok := safeConfiguredStaticDir(dir); ok {
		t.Fatalf("safeConfiguredStaticDir(%q) = %q, true; want rejected without index", dir, got)
	}
}

func TestSafeConfiguredStaticDirAcceptsNormalStaticDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := safeConfiguredStaticDir(dir)
	if !ok {
		t.Fatal("expected static dir with index.html to be accepted")
	}
	if got != want {
		t.Fatalf("safeConfiguredStaticDir returned %q, want %q", got, want)
	}
}
