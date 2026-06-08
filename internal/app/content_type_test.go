package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequireJSONContentTypeRejectsTextPlain(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{"prompt":"do work"}`))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	if requireJSONContentType(rr, req) {
		t.Fatal("text/plain was accepted as JSON")
	}
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnsupportedMediaType)
	}
}

func TestRequireJSONContentTypeAcceptsJSONWithCharset(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{"prompt":"do work"}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rr := httptest.NewRecorder()
	if !requireJSONContentType(rr, req) {
		t.Fatal("application/json with charset was rejected")
	}
}
