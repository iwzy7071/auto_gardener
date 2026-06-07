package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestAppendUsageEventRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	dataDir := t.TempDir()
	forestDir := filepath.Join(dataDir, "forests", "forest_abc")
	if err := os.MkdirAll(forestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dataDir, "outside-usage.jsonl")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	usagePath := filepath.Join(forestDir, "usage.jsonl")
	if err := os.Symlink(target, usagePath); err != nil {
		t.Fatal(err)
	}
	store := &Store{dataDir: dataDir}

	store.AppendUsageEvent("forest_abc", usageLogEvent{TaskID: "forest_abc", Time: time.Now(), Line: "total tokens: 1"})
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("symlink target was appended: %q", data)
	}
}
