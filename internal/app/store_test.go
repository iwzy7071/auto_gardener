package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadResetsSchedulePathOutsideForest(t *testing.T) {
	dataDir := t.TempDir()
	forestDir := filepath.Join(dataDir, "forests", "forest_abc")
	if err := os.MkdirAll(forestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(dataDir, "outside-schedule.md")
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	forestJSON := fmt.Sprintf(`{"id":"forest_abc","status":"Running","gardenerStatus":"Running","schedulePath":%q}`, outside)
	if err := os.WriteFile(filepath.Join(forestDir, "forest.json"), []byte(forestJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := NewStore(dataDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WriteSchedule("forest_abc", "new schedule"); err != nil {
		t.Fatal(err)
	}
	outsideData, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(outsideData) != "keep" {
		t.Fatalf("outside schedule was overwritten: %q", outsideData)
	}
	inside, err := os.ReadFile(filepath.Join(forestDir, "gardener", "schedule.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(inside) != "new schedule" {
		t.Fatalf("schedule was not written to task directory: %q", inside)
	}
}
