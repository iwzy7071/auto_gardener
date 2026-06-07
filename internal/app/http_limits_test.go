package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSettingsRejectsOversizedJSONBody(t *testing.T) {
	hub := NewEventHub()
	store, err := NewStore(t.TempDir(), hub)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, nil, t.TempDir(), hub)
	body := `{"logLevel":"` + strings.Repeat("x", int(maxSettingsJSONBodyBytes)+1) + `"}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(body))
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusRequestEntityTooLarge, rr.Body.String())
	}
}
