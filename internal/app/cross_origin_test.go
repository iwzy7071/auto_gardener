package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestHasSameOriginRejectsCrossSiteFetchMetadata(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/api/tasks", nil)
	req.Host = "127.0.0.1:8080"
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	if requestHasSameOrigin(req) {
		t.Fatal("cross-site Fetch Metadata request should be rejected")
	}
}

func TestRequestHasSameOriginAllowsSameOriginFetchMetadata(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/api/tasks", nil)
	req.Host = "127.0.0.1:8080"
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	if !requestHasSameOrigin(req) {
		t.Fatal("same-origin Fetch Metadata request should be allowed")
	}
}
