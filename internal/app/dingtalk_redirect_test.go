package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDingTalkHTTPClientDoesNotFollowRedirects(t *testing.T) {
	targetCalled := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetCalled = true
	}))
	defer target.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirector.Close()
	resp, err := newDingTalkHTTPClient().Post(redirector.URL, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusFound)
	}
	if targetCalled {
		t.Fatal("DingTalk client followed redirect to target")
	}
}
