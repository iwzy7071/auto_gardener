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

func TestClaudeModelNameLengthLimit(t *testing.T) {
	args := appendClaudeModelArg(nil, strings.Repeat("a", maxClaudeModelNameLength+1))
	if len(args) != 0 {
		t.Fatalf("appendClaudeModelArg accepted oversized model name: %#v", args)
	}

	model := strings.Repeat("a", maxClaudeModelNameLength)
	args = appendClaudeModelArg(nil, model)
	if len(args) != 2 || args[0] != "--model" || args[1] != model {
		t.Fatalf("appendClaudeModelArg rejected boundary model name: %#v", args)
	}
}
