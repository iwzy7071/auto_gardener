package codex

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
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

func TestValidateCodexCommandFromEnv(t *testing.T) {
	dir := t.TempDir()
	name := "fake-codex"
	body := "#!/bin/sh\necho codex-cli test\n"
	if runtime.GOOS == "windows" {
		name += ".bat"
		body = "@echo off\r\necho codex-cli test\r\n"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0755); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	t.Setenv("AUTO_GARDENER_RUNNER", "")
	t.Setenv("AUTO_GARDENER_CODEX_CMD", path)

	got, err := ValidateCodexCommandFromEnv(context.Background())
	if err != nil {
		t.Fatalf("ValidateCodexCommandFromEnv: %v", err)
	}
	if got != path {
		t.Fatalf("resolved command = %q, want %q", got, path)
	}
}

func TestValidateCodexCommandFromEnvFailsMissingCommand(t *testing.T) {
	t.Setenv("AUTO_GARDENER_RUNNER", "")
	t.Setenv("AUTO_GARDENER_CODEX_CMD", filepath.Join(t.TempDir(), "missing-codex"))

	if _, err := ValidateCodexCommandFromEnv(context.Background()); err == nil {
		t.Fatal("ValidateCodexCommandFromEnv succeeded for missing command")
	}
}
