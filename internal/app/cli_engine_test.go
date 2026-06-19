package app

import "testing"

func TestNormalizeCLIEngineAliases(t *testing.T) {
	cases := map[CLIEngine]CLIEngine{
		"codex":       CLIEngineCodex,
		"Codex-CLI":   CLIEngineCodex,
		"openai":      CLIEngineCodex,
		"claude":      CLIEngineClaude,
		"Claude_Code": CLIEngineClaude,
		"anthropic":   CLIEngineClaude,
		"cloud":       CLIEngineClaude,
		"":            CLIEngineCodex,
		"unknown":     CLIEngineCodex,
	}
	for in, want := range cases {
		if got := normalizeCLIEngine(in); got != want {
			t.Fatalf("normalizeCLIEngine(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCompatibleCLIEngineForBundledProviders(t *testing.T) {
	if got := compatibleCLIEngine(CLIEngineCodex, ModelModeKimi); got != CLIEngineClaude {
		t.Fatalf("Kimi should use Claude-compatible path, got %q", got)
	}
	if got := compatibleCLIEngine(CLIEngineCodex, ModelModeMiniMax); got != CLIEngineClaude {
		t.Fatalf("MiniMax should use Claude-compatible path, got %q", got)
	}
	if got := compatibleCLIEngine(CLIEngineClaude, ModelModeDefault); got != CLIEngineClaude {
		t.Fatalf("Claude default should stay Claude, got %q", got)
	}
}
