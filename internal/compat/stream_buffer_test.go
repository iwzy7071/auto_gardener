package compat

import (
	"strings"
	"testing"
)

func TestAppendLimitedStreamBufferCapsText(t *testing.T) {
	var out strings.Builder
	appendLimitedStreamBuffer(&out, "abcd", 6)
	appendLimitedStreamBuffer(&out, "efgh", 6)
	if out.Len() != 6 {
		t.Fatalf("buffer length = %d, want 6", out.Len())
	}
	appendLimitedStreamBuffer(&out, "ignored", 6)
	if out.Len() != 6 {
		t.Fatalf("buffer grew past limit: %d", out.Len())
	}
}

func TestAppendLimitedStreamBufferKeepsChunkWhenRoom(t *testing.T) {
	var out strings.Builder
	appendLimitedStreamBuffer(&out, "hello", 16)
	if got := out.String(); got != "hello" {
		t.Fatalf("buffer = %q, want hello", got)
	}
}
