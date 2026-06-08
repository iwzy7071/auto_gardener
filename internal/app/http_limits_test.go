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

func TestDecodeLimitedJSONRejectsTrailingValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{"prompt":"do work"} {}`))
	rr := httptest.NewRecorder()
	var dst CreateTaskRequest
	if decodeLimitedJSON(rr, req, &dst, maxTaskJSONBodyBytes, "bad json") {
		t.Fatal("decodeLimitedJSON accepted trailing JSON value")
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}
