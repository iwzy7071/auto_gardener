package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseUsageJSONLLimitsRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxUsageRecordsPerFile+25; i++ {
		ev := usageLogEvent{Time: time.Unix(int64(i), 0), TaskID: "task", RunID: "", SourceType: "tree", SourceID: "x", SourceName: "Tree", Line: "total tokens: 1"}
		b, err := json.Marshal(ev)
		if err != nil {
			_ = f.Close()
			t.Fatal(err)
		}
		if _, err := f.Write(append(b, '\n')); err != nil {
			_ = f.Close()
			t.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	records := parseUsageJSONL(path)
	if len(records) != maxUsageRecordsPerFile {
		t.Fatalf("records = %d, want %d", len(records), maxUsageRecordsPerFile)
	}
}

func TestParseLegacyUsageFileLimitsRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "progress.log")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxUsageRecordsPerFile+25; i++ {
		if _, err := f.WriteString("[2026-01-01T00:00:00Z] stderr: total tokens: 1\n"); err != nil {
			_ = f.Close()
			t.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	records := parseLegacyUsageFile("task", path, "tree", "x", "Tree")
	if len(records) != maxUsageRecordsPerFile {
		t.Fatalf("records = %d, want %d", len(records), maxUsageRecordsPerFile)
	}
}
