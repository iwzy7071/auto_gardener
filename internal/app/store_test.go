package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestLoadRecoveryRejectsSymlinkProgressLog(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	dataDir := t.TempDir()
	forestDir := filepath.Join(dataDir, "forests", "forest_abc")
	treeDir := filepath.Join(forestDir, "trees", "tree_abc")
	if err := os.MkdirAll(treeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(forestDir, "forest.json"), []byte(`{"id":"forest_abc","status":"Running","gardenerStatus":"Running"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	treeJSON := `{"id":"tree_abc","taskId":"forest_abc","status":"Running","updatedAt":"` + time.Now().Format(time.RFC3339) + `"}`
	if err := os.WriteFile(filepath.Join(treeDir, "tree.json"), []byte(treeJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dataDir, "outside-progress.log")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(treeDir, "progress.log")); err != nil {
		t.Fatal(err)
	}

	if _, err := NewStore(dataDir, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("symlink target was appended during recovery: %q", data)
	}
}
