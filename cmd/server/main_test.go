package main

import (
	"strings"
	"testing"
)

func TestSanitizeStartupLogValueEscapesControlCharacters(t *testing.T) {
	got := sanitizeStartupLogValue("codex\nWARN forged\r\t\x01\x7f")
	if strings.ContainsAny(got, "\n\r\t\x01\x7f") {
		t.Fatalf("sanitizeStartupLogValue left raw control characters: %q", got)
	}
	want := `codex\nWARN forged\r\t\x01\x7f`
	if got != want {
		t.Fatalf("sanitizeStartupLogValue() = %q, want %q", got, want)
	}
}

func TestSanitizeStartupLogValueKeepsPlainValue(t *testing.T) {
	value := "/usr/local/bin/codex"
	if got := sanitizeStartupLogValue(value); got != value {
		t.Fatalf("sanitizeStartupLogValue() = %q, want %q", got, value)
	}
}
