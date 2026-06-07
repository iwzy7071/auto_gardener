package app

import (
	"strings"
	"testing"
)

func TestValidateUserTextSizeRejectsOversizedText(t *testing.T) {
	if err := validateUserTextSize("消息", strings.Repeat("x", maxUserTextBytes+1)); err == nil {
		t.Fatal("oversized text accepted")
	}
}

func TestValidateUserTextSizeAllowsLimit(t *testing.T) {
	if err := validateUserTextSize("消息", strings.Repeat("x", maxUserTextBytes)); err != nil {
		t.Fatalf("text at limit rejected: %v", err)
	}
}
