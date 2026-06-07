package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadProgressUsesTailForLargeLogs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.log")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("first-line\n" + strings.Repeat("x", int(maxStartupLogReadBytes)+32) + "\nlast-line\n"); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	lines := readProgress(path)
	if len(lines) == 0 || lines[len(lines)-1] != "last-line" {
		t.Fatalf("tail progress missing last line: %#v", lines)
	}
	for _, line := range lines {
		if line == "first-line" {
			t.Fatalf("readProgress read from beginning of oversized log: %#v", lines)
		}
	}
}

func TestReadLogTailCapsBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.md")
	if err := os.WriteFile(path, []byte(strings.Repeat("a", 128)), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readLogTail(path, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 8 {
		t.Fatalf("tail length = %d, want 8", len(got))
	}
}
