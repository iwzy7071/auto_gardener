package app

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogRequestsSanitizesPathNewlines(t *testing.T) {
	var buf bytes.Buffer
	oldOutput := log.Writer()
	oldFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(oldOutput)
		log.SetFlags(oldFlags)
	}()

	handler := logRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/%0aFAKE", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	out := buf.String()
	if strings.Contains(out, "\nFAKE") || strings.Count(out, "\n") != 1 {
		t.Fatalf("request log was not kept to one line: %q", out)
	}
	if !strings.Contains(out, "GET /api/ FAKE") {
		t.Fatalf("request log = %q, want sanitized path", out)
	}
}
