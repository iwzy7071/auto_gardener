package main

import "testing"

func TestIsExternalBind(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:8080", false},
		{"localhost:8080", false},
		{"[::1]:8080", false},
		{":8080", true},
		{"0.0.0.0:8080", true},
		{"[::]:8080", true},
		{"192.168.1.20:8080", true},
		{"gardener.example.com:8080", true},
	}
	for _, tt := range tests {
		if got := isExternalBind(tt.addr); got != tt.want {
			t.Fatalf("isExternalBind(%q) = %v, want %v", tt.addr, got, tt.want)
		}
	}
}

func TestNewHTTPServerHasHeaderTimeout(t *testing.T) {
	server := newHTTPServer("127.0.0.1:0", nil)
	if server.ReadHeaderTimeout <= 0 {
		t.Fatal("ReadHeaderTimeout must be set")
	}
}
