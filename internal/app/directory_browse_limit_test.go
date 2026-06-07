package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleDirectoryBrowseLimitsReturnedDirs(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < maxDirectoryBrowseDirs+25; i++ {
		if err := os.Mkdir(filepath.Join(root, fmt.Sprintf("dir-%04d", i)), 0700); err != nil {
			t.Fatal(err)
		}
	}

	store := &Store{tasks: make(map[string]*Task), dataDir: t.TempDir(), events: NewEventHub(), settings: defaultSettings()}
	server := NewServer(store, nil, "", NewEventHub())
	req := httptest.NewRequest(http.MethodGet, "/api/fs/dirs?path="+root, nil)
	rec := httptest.NewRecorder()

	server.handleDirectoryBrowse(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body struct {
		Entries []directoryEntry `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Entries) != maxDirectoryBrowseDirs {
		t.Fatalf("entries = %d, want %d", len(body.Entries), maxDirectoryBrowseDirs)
	}
}
