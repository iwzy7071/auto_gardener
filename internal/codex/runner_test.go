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

func TestSanitizedRunnerEnvStripsRelaySecrets(t *testing.T) {
	env := sanitizedRunnerEnv([]string{
		"GARDENER_RELAY_SETUP_KEY=sk_secret",
		"GARDENER_RELAY_BASE_URL=https://relay.example.invalid",
		"PATH=/usr/bin",
	})
	for _, item := range env {
		if strings.HasPrefix(item, "GARDENER_RELAY_") {
			t.Fatalf("relay secret leaked into runner env: %s", item)
		}
	}
	if !containsString(env, "PATH=/usr/bin") {
		t.Fatalf("non-secret env was not preserved: %#v", env)
	}
}
