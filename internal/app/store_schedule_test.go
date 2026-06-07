package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppendScheduleFileAddsSeparatorWithoutReadingWholeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schedule.md")
	if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := appendScheduleFile(path, "\n## next\n"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "existing\n\n## next\n"
	if string(got) != want {
		t.Fatalf("schedule = %q, want %q", string(got), want)
	}
}
