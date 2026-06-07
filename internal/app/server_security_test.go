package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleDirectoryBrowseRejectsLongPath(t *testing.T) {
	s := &Server{}
	path := strings.Repeat("a", maxDirectoryBrowsePathLength+1)
	req := httptest.NewRequest(http.MethodGet, "/api/fs/dirs?path="+path, nil)
	rr := httptest.NewRecorder()
	s.handleDirectoryBrowse(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for long directory path, got %d", rr.Code)
	}
}
