package app

import (
	"strings"
	"testing"
)

func TestCodexLogSummaryRemovesLineBreaks(t *testing.T) {
	got := codexLogSummary("第一行\n伪造记录\t继续", 1200)
	if strings.ContainsAny(got, "\r\n\t") {
		t.Fatalf("summary contains control whitespace: %q", got)
	}
	if got != "第一行 伪造记录 继续" {
		t.Fatalf("summary = %q", got)
	}
}

func TestCodexLogSummaryTruncates(t *testing.T) {
	got := codexLogSummary("abcdef", 3)
	if got != "abc..." {
		t.Fatalf("summary = %q, want %q", got, "abc...")
	}
}
