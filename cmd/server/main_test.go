package main

import (
	"net/http"
	"testing"
	"time"
)

func TestNewHTTPServerSetsIdleTimeout(t *testing.T) {
	server := newHTTPServer("127.0.0.1:0", http.NewServeMux())
	if server.IdleTimeout != 60*time.Second {
		t.Fatalf("IdleTimeout = %s, want %s", server.IdleTimeout, 60*time.Second)
	}
}
