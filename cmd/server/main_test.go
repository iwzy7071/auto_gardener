package main

import (
	"strings"
	"testing"
)

func TestGetenvLimitedRejectsLongValue(t *testing.T) {
	t.Setenv("AUTO_GARDENER_TEST_ADDR", "127.0.0.1:8080")
	if got := getenvLimited("AUTO_GARDENER_TEST_ADDR", "fallback", maxListenAddrLength); got != "127.0.0.1:8080" {
		t.Fatalf("getenvLimited valid = %q", got)
	}
	t.Setenv("AUTO_GARDENER_TEST_ADDR", strings.Repeat("a", maxListenAddrLength+1))
	if got := getenvLimited("AUTO_GARDENER_TEST_ADDR", "fallback", maxListenAddrLength); got != "fallback" {
		t.Fatalf("getenvLimited over-limit = %q, want fallback", got)
	}
}
