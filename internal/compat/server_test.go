package compat

import (
	"net/http"
	"testing"
	"time"
)

func TestNewProxyServerSetsIdleTimeout(t *testing.T) {
	server := newProxyServer(http.NewServeMux())
	if server.IdleTimeout != 60*time.Second {
		t.Fatalf("IdleTimeout = %s, want %s", server.IdleTimeout, 60*time.Second)
	}
}
