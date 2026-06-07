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

func TestModelProviderNameLengthLimit(t *testing.T) {
	model := ModelConfig{
		ProviderID:   "custom",
		ProviderName: strings.Repeat("a", maxModelProviderNameLength+1),
		Model:        "model",
		BaseURL:      "https://example.test/v1",
		EnvKey:       "CUSTOM_API_KEY",
	}
	args := strings.Join(appendModelArgs(nil, model), "\n")
	if strings.Contains(args, ".name=") {
		t.Fatalf("appendModelArgs accepted oversized provider name in:\n%s", args)
	}

	model.ProviderName = strings.Repeat("a", maxModelProviderNameLength)
	args = strings.Join(appendModelArgs(nil, model), "\n")
	if !strings.Contains(args, ".name=") {
		t.Fatalf("appendModelArgs rejected boundary provider name in:\n%s", args)
	}
}
