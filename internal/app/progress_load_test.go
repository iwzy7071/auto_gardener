package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadProgressLoadsOnlyTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.log")
	var b strings.Builder
	b.WriteString(strings.Repeat("old\n", int(maxProgressLogLoadBytes/4)))
	for i := 0; i < 90; i++ {
		fmt.Fprintf(&b, "new-%02d\n", i)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	lines := readProgress(path)
	if len(lines) != 80 {
		t.Fatalf("expected 80 tail lines, got %d", len(lines))
	}
	if lines[0] != "new-10" || lines[len(lines)-1] != "new-89" {
		t.Fatalf("unexpected progress tail: first=%q last=%q", lines[0], lines[len(lines)-1])
	}
}

func TestReadGardenerProgressLoadsOnlyTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.md")
	var b strings.Builder
	b.WriteString(strings.Repeat("old gardener line\n", int(maxProgressLogLoadBytes/18)))
	base := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	for i := 0; i < 90; i++ {
		fmt.Fprintf(&b, "[%s] recent-%02d\n", base.Add(time.Duration(i)*time.Second).Format(time.RFC3339), i)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	lines := readGardenerProgress(path)
	if len(lines) != 80 {
		t.Fatalf("expected 80 tail lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "recent-10") || !strings.Contains(lines[len(lines)-1], "recent-89") {
		t.Fatalf("unexpected gardener progress tail: first=%q last=%q", lines[0], lines[len(lines)-1])
	}
}
