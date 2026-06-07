package app

import (
	"strings"
	"testing"
)

func TestCompactLogValueRemovesLineBreaks(t *testing.T) {
	got := compactLogValue("第一行\n伪造记录\t继续", 0)
	if strings.ContainsAny(got, "\r\n\t") {
		t.Fatalf("log value contains control whitespace: %q", got)
	}
	if got != "第一行 伪造记录 继续" {
		t.Fatalf("log value = %q", got)
	}
}

func TestCompactLogValueTruncatesLongInput(t *testing.T) {
	got := compactLogValue("abcdef", 3)
	if got != "abc..." {
		t.Fatalf("log value = %q, want %q", got, "abc...")
	}
}
