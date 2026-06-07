package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHasLegacyInterruptedRunScansBoundedTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.md")
	marker := "服务重启后发现上次遗留 Running 状态，已恢复为 Finished"
	content := strings.Repeat("old log line\n", int(maxLegacyLogScanBytes/13)) + "\n" + marker + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if !hasLegacyInterruptedRun(path) {
		t.Fatalf("expected marker in bounded tail to be detected")
	}
	tail, err := readLegacyLogTail(path)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(tail)) > maxLegacyLogScanBytes {
		t.Fatalf("legacy log scan was not bounded: %d", len(tail))
	}
}

func TestHasLegacyInterruptedRunIgnoresMarkerOutsideBoundedTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.md")
	marker := "服务重启后发现上次遗留 Running 状态，已恢复为 Finished"
	content := marker + "\n" + strings.Repeat("new log line\n", int(maxLegacyLogScanBytes/12)+100)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if hasLegacyInterruptedRun(path) {
		t.Fatalf("old marker outside bounded tail should not force legacy recovery")
	}
}
