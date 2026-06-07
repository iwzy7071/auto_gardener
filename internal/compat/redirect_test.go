package compat

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCompatHTTPClientRejectsCrossHostRedirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("cross-host redirect target received request")
	}))
	defer target.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirector.Close()
	client := newCompatHTTPClient()
	req, err := http.NewRequest(http.MethodGet, redirector.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer secret")
	resp, err := client.Do(req)
	if err == nil {
		if resp != nil {
			resp.Body.Close()
		}
		t.Fatal("cross-host redirect was allowed")
	}
}

func TestSameRedirectHostRequiresSameHostAndPort(t *testing.T) {
	if !sameRedirectHost(mustParseURL(t, "https://api.example.com/a"), mustParseURL(t, "https://api.example.com/b")) {
		t.Fatal("same host should be allowed")
	}
	if sameRedirectHost(mustParseURL(t, "https://api.example.com:8443/a"), mustParseURL(t, "https://api.example.com:443/b")) {
		t.Fatal("different port should not be allowed")
	}
	if sameRedirectHost(mustParseURL(t, "https://evil.example/a"), mustParseURL(t, "https://api.example.com/b")) {
		t.Fatal("different hostname should not be allowed")
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
