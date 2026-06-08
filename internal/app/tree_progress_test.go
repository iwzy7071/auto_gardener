package app

import (
	"errors"
	"strings"
	"testing"
)

func TestTreeErrorProgressLineSanitizesError(t *testing.T) {
	got := treeErrorProgressLine(errors.New("第一行\n伪造进度\t继续"))
	if strings.ContainsAny(got, "\r\n\t") {
		t.Fatalf("progress line contains control whitespace: %q", got)
	}
	if !strings.Contains(got, "第一行 伪造进度 继续") {
		t.Fatalf("progress line = %q, want compacted error", got)
	}
}

func TestTreeErrorProgressLineTruncatesError(t *testing.T) {
	got := treeErrorProgressLine(errors.New(strings.Repeat("x", 600)))
	if len([]rune(got)) > 503 {
		t.Fatalf("progress line was not truncated: length %d", len([]rune(got)))
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("progress line = %q, want truncation suffix", got)
	}
}
