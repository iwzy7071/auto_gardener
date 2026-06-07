package codex

import (
	"strings"
	"testing"
)

func TestAppendLimitedOutputCapsBuffer(t *testing.T) {
	var out strings.Builder
	appendLimitedOutput(&out, "abcd", 6)
	appendLimitedOutput(&out, "efgh", 6)
	if out.Len() != 6 {
		t.Fatalf("buffer length = %d, want 6", out.Len())
	}
	appendLimitedOutput(&out, "ignored", 6)
	if out.Len() != 6 {
		t.Fatalf("buffer grew past limit: %d", out.Len())
	}
}

func TestAppendLimitedOutputKeepsNewlineWhenRoom(t *testing.T) {
	var out strings.Builder
	appendLimitedOutput(&out, "line", 16)
	if got := out.String(); got != "line\n" {
		t.Fatalf("buffer = %q, want line with newline", got)
	}
}
