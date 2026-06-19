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

func TestRedactSensitiveTextRedactsModelToken(t *testing.T) {
	got := redactSensitiveText("failed with token sk-secret-token-value", ModelConfig{Token: "sk-secret-token-value"})
	if strings.Contains(got, "sk-secret-token-value") {
		t.Fatalf("token was not redacted: %q", got)
	}
	if !strings.Contains(got, "[redacted-token]") {
		t.Fatalf("redaction marker missing: %q", got)
	}
}

func TestClaudeModelArgUsesKimiModel(t *testing.T) {
	t.Setenv("AUTO_GARDENER_CLAUDE_MODEL", "")
	got := claudeModelArg(ModelConfig{ProviderID: "gardener-kimi", Model: "kimi-k2.7-code"})
	if got != "kimi-k2.7-code" {
		t.Fatalf("claudeModelArg = %q, want kimi-k2.7-code", got)
	}
}

func TestClaudeModelArgEnvOverrideWins(t *testing.T) {
	t.Setenv("AUTO_GARDENER_CLAUDE_MODEL", "custom-claude-model")
	got := claudeModelArg(ModelConfig{ProviderID: "gardener-kimi", Model: "kimi-k2.7-code"})
	if got != "custom-claude-model" {
		t.Fatalf("claudeModelArg override = %q, want custom-claude-model", got)
	}
}

func TestClaudeModelArgUsesMiniMaxModel(t *testing.T) {
	t.Setenv("AUTO_GARDENER_CLAUDE_MODEL", "")
	got := claudeModelArg(ModelConfig{ProviderID: "gardener-minimax", Model: "MiniMax-M3"})
	if got != "MiniMax-M3" {
		t.Fatalf("claudeModelArg = %q, want MiniMax-M3", got)
	}
}

func TestAppendClaudeEnvMiniMaxAnthropicEndpoint(t *testing.T) {
	t.Setenv("AUTO_GARDENER_CLAUDE_SETTINGS", "/no/such/claude-settings.json")
	t.Setenv("AUTO_GARDENER_MINIMAX_CLAUDE_BASE_URL", "")
	t.Setenv("AUTO_GARDENER_MINIMAX_CLAUDE_MODEL", "")
	env := appendClaudeEnv([]string{"ANTHROPIC_BASE_URL=https://old.example"}, ModelConfig{ProviderID: "gardener-minimax", Model: "MiniMax-M3", Token: "sk-test-token"})
	if got := testEnvValue(env, "ANTHROPIC_BASE_URL"); got != "https://api.minimaxi.com/anthropic/" {
		t.Fatalf("ANTHROPIC_BASE_URL = %q", got)
	}
	if got := testEnvValue(env, "ANTHROPIC_AUTH_TOKEN"); got != "sk-test-token" {
		t.Fatalf("ANTHROPIC_AUTH_TOKEN = %q", got)
	}
	if got := testEnvValue(env, "ANTHROPIC_API_KEY"); got != "sk-test-token" {
		t.Fatalf("ANTHROPIC_API_KEY fallback = %q", got)
	}
	for _, key := range []string{"ANTHROPIC_MODEL", "ANTHROPIC_DEFAULT_OPUS_MODEL", "ANTHROPIC_DEFAULT_SONNET_MODEL", "ANTHROPIC_DEFAULT_HAIKU_MODEL", "CLAUDE_CODE_SUBAGENT_MODEL"} {
		if got := testEnvValue(env, key); got != "MiniMax-M3" {
			t.Fatalf("%s = %q", key, got)
		}
	}
	if got := testEnvValue(env, "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"); got != "1" {
		t.Fatalf("CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC = %q", got)
	}
	if got := testEnvValue(env, "API_TIMEOUT_MS"); got != "300000" {
		t.Fatalf("API_TIMEOUT_MS = %q", got)
	}
}

func TestAppendClaudeEnvUsesOfficialKimiAnthropicEndpoint(t *testing.T) {
	t.Setenv("AUTO_GARDENER_CLAUDE_SETTINGS", "/no/such/claude-settings.json")
	t.Setenv("AUTO_GARDENER_KIMI_CLAUDE_BASE_URL", "")
	t.Setenv("AUTO_GARDENER_KIMI_CLAUDE_MODEL", "")
	env := appendClaudeEnv([]string{"ANTHROPIC_BASE_URL=https://old.example"}, ModelConfig{ProviderID: "gardener-kimi", Model: "kimi-k2.7-code", Token: "sk-test-token"})
	if got := testEnvValue(env, "ANTHROPIC_BASE_URL"); got != "https://api.kimi.com/coding/" {
		t.Fatalf("ANTHROPIC_BASE_URL = %q", got)
	}
	if got := testEnvValue(env, "ANTHROPIC_AUTH_TOKEN"); got != "sk-test-token" {
		t.Fatalf("ANTHROPIC_AUTH_TOKEN = %q", got)
	}
	if got := testEnvValue(env, "ANTHROPIC_API_KEY"); got != "sk-test-token" {
		t.Fatalf("ANTHROPIC_API_KEY fallback = %q", got)
	}
	for _, key := range []string{"ANTHROPIC_MODEL", "ANTHROPIC_DEFAULT_OPUS_MODEL", "ANTHROPIC_DEFAULT_SONNET_MODEL", "ANTHROPIC_DEFAULT_HAIKU_MODEL", "CLAUDE_CODE_SUBAGENT_MODEL"} {
		if got := testEnvValue(env, key); got != "kimi-k2.7-code" {
			t.Fatalf("%s = %q", key, got)
		}
	}
	if got := testEnvValue(env, "ENABLE_TOOL_SEARCH"); got != "false" {
		t.Fatalf("ENABLE_TOOL_SEARCH = %q", got)
	}
	if got := testEnvValue(env, "CLAUDE_CODE_AUTO_COMPACT_WINDOW"); got != "262144" {
		t.Fatalf("CLAUDE_CODE_AUTO_COMPACT_WINDOW = %q", got)
	}
	if got := testEnvValue(env, "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"); got != "1" {
		t.Fatalf("CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC = %q", got)
	}
	if got := testEnvValue(env, "API_TIMEOUT_MS"); got != "300000" {
		t.Fatalf("API_TIMEOUT_MS = %q", got)
	}
}

func TestAppendClaudeEnvKimiOverrides(t *testing.T) {
	t.Setenv("AUTO_GARDENER_CLAUDE_SETTINGS", "/no/such/claude-settings.json")
	t.Setenv("AUTO_GARDENER_KIMI_CLAUDE_BASE_URL", "https://override.example/anthropic")
	t.Setenv("AUTO_GARDENER_KIMI_CLAUDE_MODEL", "custom-kimi")
	t.Setenv("AUTO_GARDENER_KIMI_ENABLE_TOOL_SEARCH", "true")
	t.Setenv("AUTO_GARDENER_KIMI_AUTO_COMPACT_WINDOW", "12345")
	env := appendClaudeEnv(nil, ModelConfig{ProviderID: "gardener-kimi", Model: "kimi-k2.7-code", Token: "sk-test-token"})
	if got := testEnvValue(env, "ANTHROPIC_BASE_URL"); got != "https://override.example/anthropic" {
		t.Fatalf("ANTHROPIC_BASE_URL override = %q", got)
	}
	if got := testEnvValue(env, "ANTHROPIC_MODEL"); got != "custom-kimi" {
		t.Fatalf("ANTHROPIC_MODEL override = %q", got)
	}
	if got := testEnvValue(env, "ENABLE_TOOL_SEARCH"); got != "true" {
		t.Fatalf("ENABLE_TOOL_SEARCH override = %q", got)
	}
	if got := testEnvValue(env, "CLAUDE_CODE_AUTO_COMPACT_WINDOW"); got != "12345" {
		t.Fatalf("CLAUDE_CODE_AUTO_COMPACT_WINDOW override = %q", got)
	}
}

func TestWithProviderExecutionGuardForBundledProviders(t *testing.T) {
	prompt := "do work"
	guarded := withProviderExecutionGuard(prompt, ModelConfig{ProviderID: "gardener-minimax", Model: "MiniMax-M3"})
	if !strings.Contains(guarded, "Provider execution guard") {
		t.Fatalf("expected provider execution guard, got %q", guarded)
	}
	if !strings.Contains(guarded, "Never print raw tool-call markup") {
		t.Fatalf("guard missing raw tool-call instruction: %q", guarded)
	}
	if !strings.HasSuffix(guarded, prompt) {
		t.Fatalf("guard should preserve prompt suffix: %q", guarded)
	}
	plain := withProviderExecutionGuard(prompt, ModelConfig{ProviderID: "openai", Model: "gpt"})
	if plain != prompt {
		t.Fatalf("non-bundled provider prompt changed: %q", plain)
	}
}

func TestLeakedProviderToolCallDetectedForBundledProviders(t *testing.T) {
	output := `]<]minimax[>[<tool_call>
]<]minimax[>[<invoke name="Write">]<]minimax[>[<file_path>/tmp/out.txt</file_path>]`
	if !leakedProviderToolCall(output, ModelConfig{ProviderID: "gardener-minimax", Model: "MiniMax-M3"}) {
		t.Fatal("expected MiniMax raw tool-call markup to be treated as a provider failure")
	}
	if leakedProviderToolCall(output, ModelConfig{ProviderID: "openai", Model: "gpt"}) {
		t.Fatal("non-bundled providers should not be classified by this guard")
	}
}

func testEnvValue(env []string, key string) string {
	for _, item := range env {
		k, v, ok := strings.Cut(item, "=")
		if ok && k == key {
			return v
		}
	}
	return ""
}
