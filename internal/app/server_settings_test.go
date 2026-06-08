package app

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleSettingsUpdateHidesPersistPath(t *testing.T) {
	tmp := t.TempDir()
	secretName := "secret-settings-dir"
	blockedPath := filepath.Join(tmp, secretName)
	if err := os.WriteFile(blockedPath, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}

	store := &Store{tasks: make(map[string]*Task), dataDir: blockedPath, events: NewEventHub(), settings: defaultSettings()}
	server := NewServer(store, nil, "", NewEventHub())

	req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewBufferString(`{"logLevel":"normal"}`))
	rec := httptest.NewRecorder()

	server.handleSettings(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	body := rec.Body.String()
	if strings.Contains(body, secretName) || strings.Contains(body, blockedPath) || strings.Contains(body, "settings.json") {
		t.Fatalf("response leaked persistence path: %s", body)
	}
	if !strings.Contains(body, "保存设置失败") {
		t.Fatalf("response body = %s, want generic settings error", body)
	}
}
