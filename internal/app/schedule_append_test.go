package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadScheduleForAppendLimitsOversizedHistory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schedule.md")
	var b strings.Builder
	b.WriteString(strings.Repeat("old schedule line\n", int(maxScheduleAppendReadBytes/18)))
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&b, "recent schedule %02d\n", i)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readScheduleForAppend(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "schedule history truncated") {
		prefixLen := len(got)
		if prefixLen > 80 {
			prefixLen = 80
		}
		t.Fatalf("expected truncation notice, got prefix %q", got[:prefixLen])
	}
	if !strings.Contains(got, "recent schedule 19") {
		t.Fatalf("expected recent schedule content to be preserved")
	}
	if len(got) > int(maxScheduleAppendReadBytes)+200 {
		t.Fatalf("schedule append read was not bounded: %d", len(got))
	}
}

func TestReadScheduleForAppendKeepsSmallSchedule(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schedule.md")
	want := "# Schedule\n\nkeep all\n"
	if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readScheduleForAppend(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("unexpected schedule: %q", got)
	}
}
