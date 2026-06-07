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

func TestWindowsNPMDirsSkipsOverlongEnvPaths(t *testing.T) {
	longPath := "C:\\" + strings.Repeat("a", maxWindowsNPMDirLength+1)
	t.Setenv("NPM_CONFIG_PREFIX", longPath)
	dirs := windowsNPMDirs()
	for _, dir := range dirs {
		if dir == longPath {
			t.Fatalf("windowsNPMDirs included overlong path")
		}
	}
}
