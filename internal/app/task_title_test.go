package app

import (
	"strings"
	"testing"
)

func TestTitleFromPromptRemovesLineBreaks(t *testing.T) {
	title := titleFromPrompt("第一行\n伪造日志\t继续")
	if strings.ContainsAny(title, "\r\n\t") {
		t.Fatalf("title contains control whitespace: %q", title)
	}
	if title != "第一行 伪造日志 继续" {
		t.Fatalf("title = %q", title)
	}
}

func TestNormalizeTaskTitleTruncatesAfterSingleLine(t *testing.T) {
	title := normalizeTaskTitle("a\n b c", 3)
	if title != "a b" {
		t.Fatalf("title = %q, want %q", title, "a b")
	}
}
