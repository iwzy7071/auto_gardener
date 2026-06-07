package codex

import (
	"os"
	"path/filepath"
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

func TestReadRunnerOutputFileRejectsOversizedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent_last_message.md")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(make([]byte, maxRunnerOutputFileBytes+1)); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = readRunnerOutputFile(path)
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected oversized output to be rejected, got %v", err)
	}
}

func TestReadRunnerOutputFileAllowsBoundedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent_last_message.md")
	if err := os.WriteFile(path, []byte("final"), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := readRunnerOutputFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "final" {
		t.Fatalf("unexpected output: %q", b)
	}
}
