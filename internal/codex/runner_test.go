package codex

import (
	"os"
	"strings"
	"testing"
)

func TestIsClaudeCLIAliases(t *testing.T) {
	for _, in := range []string{"claude", "Claude-Code", "claude_cli", "anthropic", "cloud"} {
		if !isClaudeCLI(in) {
			t.Fatalf("isClaudeCLI(%q) = false, want true", in)
		}
	}
	for _, in := range []string{"", "codex", "openai"} {
		if isClaudeCLI(in) {
			t.Fatalf("isClaudeCLI(%q) = true, want false", in)
		}
	}
}

func TestWithGoalEnvelope(t *testing.T) {
	prompt := withGoalEnvelope("do work", GoalSpec{
		ID:              "tree_123",
		Title:           "完成子任务：实现功能",
		Objective:       "实现目标功能",
		SuccessCriteria: []string{"改动必要文件", "报告 goal 状态"},
		Path:            "/tmp/goal.md",
	})
	for _, want := range []string{"# Goal mode", "tree_123", "完成子任务：实现功能", "实现目标功能", "改动必要文件", "/tmp/goal.md", "# Task instructions", "do work"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("goal envelope missing %q in:\n%s", want, prompt)
		}
	}
}

func TestRedactSensitiveTextRedactsModelToken(t *testing.T) {
	got := redactSensitiveText("failed with token sk-secret-token-value", ModelConfig{Token: "sk-secret-token-value"})
	if strings.Contains(got, "sk-secret-token-value") {
		t.Fatalf("token was not redacted: %q", got)
	}
	if !strings.Contains(got, "[redacted-token]") {
		t.Fatalf("redaction marker missing: %q", got)
	}
}

func TestWriteOutputFileIsPrivate(t *testing.T) {
	path := t.TempDir() + "/output.txt"
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeOutputFile(path, []byte("new")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != privateOutputFileMode {
		t.Fatalf("output mode = %o, want %o", got, privateOutputFileMode)
	}
}
