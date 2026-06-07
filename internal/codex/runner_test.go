package codex

import (
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

func TestClaudeBaseURLLengthLimit(t *testing.T) {
	t.Setenv("AUTO_GARDENER_KIMI_CLAUDE_BASE_URL", "https://example.test/"+strings.Repeat("a", maxClaudeBaseURLLength))
	env := appendClaudeEnv(nil, ModelConfig{ProviderID: "gardener-kimi", Token: "secret"})
	for _, item := range env {
		if strings.HasPrefix(item, "ANTHROPIC_BASE_URL=") {
			t.Fatalf("appendClaudeEnv accepted oversized base URL: %q", item)
		}
	}

	t.Setenv("AUTO_GARDENER_KIMI_CLAUDE_BASE_URL", "https://example.test/"+strings.Repeat("a", maxClaudeBaseURLLength-len("https://example.test/")))
	env = appendClaudeEnv(nil, ModelConfig{ProviderID: "gardener-kimi", Token: "secret"})
	found := false
	for _, item := range env {
		if strings.HasPrefix(item, "ANTHROPIC_BASE_URL=") {
			found = true
		}
	}
	if !found {
		t.Fatalf("appendClaudeEnv rejected boundary base URL: %#v", env)
	}
}
