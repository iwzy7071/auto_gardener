package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseUsageJSONLIgnoresMismatchedTaskIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "usage.jsonl")
	events := []usageLogEvent{
		{Time: time.Unix(1, 0), TaskID: "other-task", RunID: "other", SourceType: "tree", SourceName: "Other", Line: "model: gpt-5"},
		{Time: time.Unix(2, 0), TaskID: "other-task", RunID: "other", SourceType: "tree", SourceName: "Other", Line: "total tokens: 999"},
		{Time: time.Unix(3, 0), TaskID: "task-a", RunID: "own", SourceType: "tree", SourceName: "Own", Line: "model: gpt-5"},
		{Time: time.Unix(4, 0), TaskID: "task-a", RunID: "own", SourceType: "tree", SourceName: "Own", Line: "total tokens: 42"},
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		b, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(append(b, '\n')); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	records := parseUsageJSONL(path, "task-a")
	if len(records) != 1 {
		t.Fatalf("expected one matching record, got %d", len(records))
	}
	if records[0].TaskID != "task-a" || records[0].TotalTokens != 42 {
		t.Fatalf("unexpected record: %#v", records[0])
	}
}
