package codex

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ModelConfig struct {
	ProviderID   string
	ProviderName string
	Model        string
	BaseURL      string
	EnvKey       string
	Token        string
	WireAPI      string
}

func (c ModelConfig) IsDefault() bool {
	return strings.TrimSpace(c.Model) == "" && strings.TrimSpace(c.ProviderID) == ""
}

type GoalSpec struct {
	ID              string
	Title           string
	Objective       string
	SuccessCriteria []string
	Path            string
}

type RunRequest struct {
	Role       string
	CLI        string
	Prompt     string
	WorkDir    string
	OutputFile string
	Model      ModelConfig
	Goal       GoalSpec
	OnLine     func(string)
}

type RunResult struct {
	Output string
	Err    error
}

type Runner interface {
	Run(ctx context.Context, req RunRequest) RunResult
}

type ShellRunner struct {
	CodexCommand  string
	ClaudeCommand string
}

func NewRunnerFromEnv() Runner {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("AUTO_GARDENER_RUNNER")), "mock") {
		r := NewMockRunnerFromEnv()
		return r
	}
	codexCmd := strings.TrimSpace(os.Getenv("AUTO_GARDENER_CODEX_CMD"))
	if codexCmd == "" {
		codexCmd = "codex"
	}
	claudeCmd := strings.TrimSpace(os.Getenv("AUTO_GARDENER_CLAUDE_CMD"))
	if claudeCmd == "" {
		claudeCmd = "claude"
	}
	return ShellRunner{CodexCommand: codexCmd, ClaudeCommand: claudeCmd}
}

func (r ShellRunner) Run(ctx context.Context, req RunRequest) RunResult {
	req.Prompt = withGoalEnvelope(req.Prompt, req.Goal)
	req.Prompt = withProviderExecutionGuard(req.Prompt, req.Model)
	if isClaudeCLI(req.CLI) {
		return r.runClaude(ctx, req)
	}
	return r.runCodex(ctx, req)
}

func isClaudeCLI(cli string) bool {
	value := strings.ToLower(strings.TrimSpace(cli))
	value = strings.ReplaceAll(value, "_", "-")
	switch value {
	case "claude", "claude-code", "claude-cli", "anthropic", "cloud":
		return true
	default:
		return false
	}
}

func withProviderExecutionGuard(prompt string, model ModelConfig) string {
	provider := strings.TrimSpace(model.ProviderID)
	if provider != "gardener-minimax" && provider != "gardener-kimi" {
		return prompt
	}
	return `# Provider execution guard

You are running inside the real Claude Code CLI through a MiniMax/Kimi compatible provider. When you need to read, write, edit, or run commands, you must trigger the CLI's actual tool execution mechanism. Never print raw tool-call markup such as <tool_call>, <invoke>, tool_use JSON/XML, or provider-tagged tool syntax in your answer. Raw tool-call markup is not executed by Gardener and will be treated as a failed run.

If the CLI refuses to execute a needed tool, stop and report Goal status: blocked with the exact reason; do not emit unexecuted tool syntax.

` + prompt
}

func withGoalEnvelope(prompt string, goal GoalSpec) string {
	if strings.TrimSpace(goal.Title) == "" && strings.TrimSpace(goal.Objective) == "" {
		return prompt
	}
	var b strings.Builder
	b.WriteString("# Goal mode\n\n")
	b.WriteString("This run is a single goal-driven agent task. If your CLI/runtime exposes native goal tracking, create/activate one goal at the start, keep it updated while working, and mark it complete or blocked at the end. If native goal tracking is unavailable, follow this goal contract exactly in your final report.\n\n")
	if strings.TrimSpace(goal.ID) != "" {
		b.WriteString("Goal ID: " + strings.TrimSpace(goal.ID) + "\n")
	}
	if strings.TrimSpace(goal.Title) != "" {
		b.WriteString("Goal title: " + strings.TrimSpace(goal.Title) + "\n")
	}
	if strings.TrimSpace(goal.Objective) != "" {
		b.WriteString("Goal objective: " + strings.TrimSpace(goal.Objective) + "\n")
	}
	if strings.TrimSpace(goal.Path) != "" {
		b.WriteString("Goal record file: " + strings.TrimSpace(goal.Path) + "\n")
	}
	if len(goal.SuccessCriteria) > 0 {
		b.WriteString("Success criteria:\n")
		for _, item := range goal.SuccessCriteria {
			item = strings.TrimSpace(item)
			if item != "" {
				b.WriteString("- " + item + "\n")
			}
		}
	}
	b.WriteString("\nRules:\n")
	b.WriteString("- Work toward this goal only; do not expand scope beyond the goal and assigned files.\n")
	b.WriteString("- If the goal is too large, unsafe, blocked, or cannot be finished in this run, stop with a clear blocked/partial status and concrete next steps instead of silently stopping.\n")
	b.WriteString("- The final answer must include a Goal status section: complete / partial / blocked, evidence, changed files, tests run, and next recommended action.\n\n")
	b.WriteString("# Task instructions\n\n")
	b.WriteString(prompt)
	return b.String()
}

func (r ShellRunner) runCodex(ctx context.Context, req RunRequest) RunResult {
	if req.WorkDir == "" {
		req.WorkDir = "."
	}
	command, env, err := resolveCommand(r.CodexCommand, "Codex CLI", "AUTO_GARDENER_CODEX_CMD")
	if err != nil {
		return RunResult{Err: err}
	}
	args := []string{
		"exec",
		"--dangerously-bypass-approvals-and-sandbox",
		"--skip-git-repo-check",
		"-C", req.WorkDir,
	}
	args = appendModelArgs(args, req.Model)
	if req.OutputFile != "" {
		args = append(args, "-o", req.OutputFile)
	}
	args = append(args, "-")
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = req.WorkDir
	cmd.Env = appendModelEnv(env, req.Model)
	cmd.Stdin = strings.NewReader(req.Prompt)
	setProcessGroup(cmd)
	cmd.Cancel = func() error { return killProcess(cmd) }
	cmd.WaitDelay = 5 * time.Second

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return RunResult{Err: err}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return RunResult{Err: err}
	}
	if err := cmd.Start(); err != nil {
		return RunResult{Err: err}
	}

	var mu sync.Mutex
	var out strings.Builder
	collect := func(prefix string, rd io.Reader) {
		s := bufio.NewScanner(rd)
		buf := make([]byte, 0, 64*1024)
		s.Buffer(buf, 1024*1024)
		for s.Scan() {
			line := redactSensitiveText(s.Text(), req.Model)
			decorated := line
			if prefix != "" {
				decorated = prefix + line
			}
			mu.Lock()
			out.WriteString(decorated)
			out.WriteByte('\n')
			mu.Unlock()
			if req.OnLine != nil {
				req.OnLine(decorated)
			}
		}
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); collect("", stdout) }()
	go func() { defer wg.Done(); collect("stderr: ", stderr) }()

	err = cmd.Wait()
	wg.Wait()
	if ctx.Err() != nil {
		_ = killProcess(cmd)
		if err == nil {
			err = ctx.Err()
		}
	}
	mu.Lock()
	output := redactSensitiveText(out.String(), req.Model)
	mu.Unlock()
	if req.OutputFile != "" {
		if b, readErr := os.ReadFile(req.OutputFile); readErr == nil && len(b) > 0 {
			// The Codex final message is the authoritative output. stdout/stderr is
			// already streamed through OnLine into log.md/progress.log. Keeping only
			// the final message here prevents Gardener JSON parsing from being polluted
			// by CLI progress text.
			output = redactSensitiveText(string(b), req.Model)
			if output != string(b) {
				_ = os.WriteFile(req.OutputFile, []byte(output), 0644)
			}
		}
	}
	return RunResult{Output: output, Err: err}
}

func (r ShellRunner) runClaude(ctx context.Context, req RunRequest) RunResult {
	if req.WorkDir == "" {
		req.WorkDir = "."
	}
	command, env, err := resolveCommand(r.ClaudeCommand, "Claude CLI", "AUTO_GARDENER_CLAUDE_CMD")
	if err != nil {
		return RunResult{Err: err}
	}
	args := []string{}
	if settingsPath := claudeSettingsPath(req.Model); settingsPath != "" {
		args = append(args, "--settings", settingsPath)
	}
	args = append(args,
		"--bare",
		"-p",
		"--output-format", "text",
		"--permission-mode", "bypassPermissions",
	)
	if model := claudeModelArg(req.Model); model != "" {
		args = append(args, "--model", model)
	}
	// Claude Code's non-interactive print mode expects the prompt as a
	// positional argument. Supplying it on stdin can leave compatible providers
	// such as Kimi waiting without issuing the intended request.
	args = append(args, req.Prompt)
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = req.WorkDir
	cmd.Env = appendClaudeEnv(env, req.Model)
	setProcessGroup(cmd)
	cmd.Cancel = func() error { return killProcess(cmd) }
	cmd.WaitDelay = 5 * time.Second

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return RunResult{Err: err}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return RunResult{Err: err}
	}
	if err := cmd.Start(); err != nil {
		return RunResult{Err: err}
	}

	var mu sync.Mutex
	var out strings.Builder
	collect := func(prefix string, rd io.Reader) {
		s := bufio.NewScanner(rd)
		buf := make([]byte, 0, 64*1024)
		s.Buffer(buf, 1024*1024)
		for s.Scan() {
			line := redactSensitiveText(s.Text(), req.Model)
			decorated := line
			if prefix != "" {
				decorated = prefix + line
			}
			mu.Lock()
			out.WriteString(decorated)
			out.WriteByte('\n')
			mu.Unlock()
			if req.OnLine != nil {
				req.OnLine(decorated)
			}
		}
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); collect("", stdout) }()
	go func() { defer wg.Done(); collect("stderr: ", stderr) }()

	err = cmd.Wait()
	wg.Wait()
	if ctx.Err() != nil {
		_ = killProcess(cmd)
		if err == nil {
			err = ctx.Err()
		}
	}
	mu.Lock()
	output := redactSensitiveText(out.String(), req.Model)
	mu.Unlock()
	if req.OutputFile != "" && strings.TrimSpace(output) != "" {
		_ = os.MkdirAll(filepath.Dir(req.OutputFile), 0755)
		_ = os.WriteFile(req.OutputFile, []byte(output), 0644)
	}
	if err == nil && leakedProviderToolCall(output, req.Model) {
		err = fmt.Errorf("provider emitted raw tool-call markup instead of executing tools")
	}
	return RunResult{Output: output, Err: err}
}

func leakedProviderToolCall(output string, model ModelConfig) bool {
	provider := strings.TrimSpace(model.ProviderID)
	if provider != "gardener-minimax" && provider != "gardener-kimi" {
		return false
	}
	return strings.Contains(output, "]<]minimax[>[<tool_call>") ||
		strings.Contains(output, "]<]kimi[>[<tool_call>") ||
		(strings.Contains(output, "<tool_call>") && strings.Contains(output, "<invoke name="))
}

func redactSensitiveText(text string, model ModelConfig) string {
	token := strings.TrimSpace(model.Token)
	if len(token) < 8 {
		return text
	}
	return strings.ReplaceAll(text, token, "[redacted-token]")
}

func appendModelArgs(args []string, model ModelConfig) []string {
	if model.IsDefault() {
		return args
	}
	if value := strings.TrimSpace(model.Model); value != "" {
		args = append(args, "-m", value)
	}
	providerID := strings.TrimSpace(model.ProviderID)
	if providerID == "" {
		return args
	}
	args = append(args, "-c", "model_provider="+tomlString(providerID))
	prefix := "model_providers." + providerID + "."
	if value := strings.TrimSpace(model.ProviderName); value != "" {
		args = append(args, "-c", prefix+"name="+tomlString(value))
	}
	if value := strings.TrimSpace(model.BaseURL); value != "" {
		args = append(args, "-c", prefix+"base_url="+tomlString(value))
	}
	if value := strings.TrimSpace(model.EnvKey); value != "" {
		args = append(args, "-c", prefix+"env_key="+tomlString(value))
	}
	wireAPI := strings.TrimSpace(model.WireAPI)
	if wireAPI == "" {
		wireAPI = "responses"
	}
	args = append(args, "-c", prefix+"wire_api="+tomlString(wireAPI))
	args = append(args, "-c", prefix+"requires_openai_auth=false")
	return args
}

func claudeModelArg(model ModelConfig) string {
	if override := strings.TrimSpace(os.Getenv("AUTO_GARDENER_CLAUDE_MODEL")); override != "" {
		return override
	}
	switch strings.TrimSpace(model.ProviderID) {
	case "gardener-kimi", "gardener-minimax":
		return strings.TrimSpace(model.Model)
	default:
		return ""
	}
}

func claudeSettingsPath(model ModelConfig) string {
	if override := strings.TrimSpace(os.Getenv("AUTO_GARDENER_CLAUDE_SETTINGS")); override != "" {
		if fileExists(override) {
			return override
		}
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	var name string
	switch strings.TrimSpace(model.ProviderID) {
	case "gardener-kimi":
		name = firstNonEmpty(os.Getenv("AUTO_GARDENER_KIMI_CLAUDE_SETTINGS_NAME"), "kimi-k2.7.settings.json")
	case "gardener-minimax":
		name = firstNonEmpty(os.Getenv("AUTO_GARDENER_MINIMAX_CLAUDE_SETTINGS_NAME"), "minimax-m3.settings.json")
	default:
		return ""
	}
	path := filepath.Join(home, ".claude", "providers", name)
	if fileExists(path) {
		return path
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func appendModelEnv(env []string, model ModelConfig) []string {
	token := strings.TrimSpace(model.Token)
	if token == "" {
		return env
	}
	envKey := strings.TrimSpace(model.EnvKey)
	if envKey != "" {
		env = upsertEnv(env, envKey, token)
	}
	switch strings.TrimSpace(model.ProviderID) {
	case "minimax", "gardener-minimax":
		env = upsertEnv(env, "MINIMAX_API_KEY", token)
	case "moonshot", "gardener-kimi":
		env = upsertEnv(env, "MOONSHOT_API_KEY", token)
		env = upsertEnv(env, "KIMI_API_KEY", token)
	}
	if usesLocalhost(model.BaseURL) {
		env = ensureNoProxy(env, "127.0.0.1")
		env = ensureNoProxy(env, "localhost")
	}
	return env
}

func appendClaudeEnv(env []string, model ModelConfig) []string {
	token := strings.TrimSpace(model.Token)
	usesSettingsFile := claudeSettingsPath(model) != ""
	switch strings.TrimSpace(model.ProviderID) {
	case "gardener-kimi":
		if token == "" && !usesSettingsFile {
			return env
		}
		modelName := firstNonEmpty(os.Getenv("AUTO_GARDENER_KIMI_CLAUDE_MODEL"), strings.TrimSpace(model.Model), "kimi-k2.7-code")
		// Kimi Coding is exposed for coding agents through its Claude-compatible
		// endpoint. Keep ANTHROPIC_API_KEY populated as a harmless compatibility
		// fallback for older Claude Code builds.
		if !usesSettingsFile {
			env = upsertEnv(env, "ANTHROPIC_BASE_URL", firstNonEmpty(os.Getenv("AUTO_GARDENER_KIMI_CLAUDE_BASE_URL"), "https://api.kimi.com/coding/"))
			env = upsertEnv(env, "ANTHROPIC_AUTH_TOKEN", token)
			env = upsertEnv(env, "ANTHROPIC_API_KEY", token)
		}
		applyClaudeModelEnv(&env, modelName)
		env = upsertEnv(env, "ENABLE_TOOL_SEARCH", firstNonEmpty(os.Getenv("AUTO_GARDENER_KIMI_ENABLE_TOOL_SEARCH"), "false"))
		env = upsertEnv(env, "CLAUDE_CODE_AUTO_COMPACT_WINDOW", firstNonEmpty(os.Getenv("AUTO_GARDENER_KIMI_AUTO_COMPACT_WINDOW"), "262144"))
		env = upsertEnv(env, "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC", firstNonEmpty(os.Getenv("AUTO_GARDENER_KIMI_DISABLE_NONESSENTIAL_TRAFFIC"), "1"))
		env = upsertEnv(env, "API_TIMEOUT_MS", firstNonEmpty(os.Getenv("AUTO_GARDENER_KIMI_CLAUDE_API_TIMEOUT_MS"), "300000"))
	case "gardener-minimax":
		if token == "" && !usesSettingsFile {
			return env
		}
		modelName := firstNonEmpty(os.Getenv("AUTO_GARDENER_MINIMAX_CLAUDE_MODEL"), strings.TrimSpace(model.Model), "MiniMax-M3")
		// MiniMax also exposes an Anthropic-compatible endpoint. This lets the app
		// keep MiniMax as the model supplier while using Claude Code as the CLI when
		// Codex-compatible planning emits invalid JSON or stalls.
		if !usesSettingsFile {
			env = upsertEnv(env, "ANTHROPIC_BASE_URL", firstNonEmpty(os.Getenv("AUTO_GARDENER_MINIMAX_CLAUDE_BASE_URL"), "https://api.minimaxi.com/anthropic/"))
			env = upsertEnv(env, "ANTHROPIC_AUTH_TOKEN", token)
			env = upsertEnv(env, "ANTHROPIC_API_KEY", token)
		}
		applyClaudeModelEnv(&env, modelName)
		env = upsertEnv(env, "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC", firstNonEmpty(os.Getenv("AUTO_GARDENER_MINIMAX_DISABLE_NONESSENTIAL_TRAFFIC"), "1"))
		env = upsertEnv(env, "API_TIMEOUT_MS", firstNonEmpty(os.Getenv("AUTO_GARDENER_MINIMAX_CLAUDE_API_TIMEOUT_MS"), "300000"))
	}
	return env
}

func applyClaudeModelEnv(env *[]string, modelName string) {
	*env = upsertEnv(*env, "ANTHROPIC_MODEL", modelName)
	*env = upsertEnv(*env, "ANTHROPIC_DEFAULT_OPUS_MODEL", modelName)
	*env = upsertEnv(*env, "ANTHROPIC_DEFAULT_SONNET_MODEL", modelName)
	*env = upsertEnv(*env, "ANTHROPIC_DEFAULT_HAIKU_MODEL", modelName)
	*env = upsertEnv(*env, "CLAUDE_CODE_SUBAGENT_MODEL", modelName)
}

func upsertEnv(env []string, key, value string) []string {
	for i, item := range env {
		k, _, ok := strings.Cut(item, "=")
		if ok && strings.EqualFold(k, key) {
			env[i] = key + "=" + value
			return env
		}
	}
	return append(env, key+"="+value)
}

func tomlString(value string) string {
	return strconv.Quote(value)
}

func firstNonEmpty(items ...string) string {
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			return item
		}
	}
	return ""
}

func usesLocalhost(rawURL string) bool {
	value := strings.ToLower(rawURL)
	return strings.Contains(value, "127.0.0.1") || strings.Contains(value, "localhost")
}

func ensureNoProxy(env []string, host string) []string {
	env = ensureNoProxyKey(env, "NO_PROXY", host)
	env = ensureNoProxyKey(env, "no_proxy", host)
	return env
}

func ensureNoProxyKey(env []string, key, host string) []string {
	for i, item := range env {
		k, value, ok := strings.Cut(item, "=")
		if ok && k == key {
			if envListContains(value, host) {
				return env
			}
			if strings.TrimSpace(value) == "" {
				env[i] = key + "=" + host
			} else {
				env[i] = key + "=" + value + "," + host
			}
			return env
		}
	}
	return append(env, key+"="+host)
}

func envListContains(value, target string) bool {
	for _, item := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(item), target) {
			return true
		}
	}
	return false
}

func resolveCommand(command, label, envVar string) (string, []string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		command = "codex"
		if strings.Contains(strings.ToLower(label), "claude") {
			command = "claude"
		}
	}
	env := os.Environ()
	if runtime.GOOS == "windows" {
		env = withWindowsNPMPath(env)
	}
	if path, err := exec.LookPath(command); err == nil {
		return path, env, nil
	}
	if filepath.IsAbs(command) || strings.ContainsAny(command, `/\`) {
		if st, err := os.Stat(command); err == nil && !st.IsDir() {
			return command, env, nil
		}
		return "", env, fmt.Errorf("找不到 %s 命令；请检查 %s 配置", label, envVar)
	}
	if runtime.GOOS == "windows" {
		if path := findWindowsNPMCommand(command); path != "" {
			env = ensurePathDir(env, filepath.Dir(path))
			return path, env, nil
		}
	}
	return "", env, fmt.Errorf("找不到 %s 命令；如果 CLI 是用 npm 在 Windows 安装，请确认 %%APPDATA%%\\npm 下存在对应 .cmd 文件，或检查 %s 配置", label, envVar)
}

func withWindowsNPMPath(env []string) []string {
	for _, dir := range windowsNPMDirs() {
		env = ensurePathDir(env, dir)
	}
	return env
}

func findWindowsNPMCommand(command string) string {
	names := []string{command}
	base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(command, ".cmd"), ".exe"), ".bat")
	for _, ext := range []string{".cmd", ".exe", ".bat"} {
		name := base + ext
		if !containsString(names, name) {
			names = append(names, name)
		}
	}
	for _, dir := range windowsNPMDirs() {
		for _, name := range names {
			path := filepath.Join(dir, name)
			if st, err := os.Stat(path); err == nil && !st.IsDir() {
				return path
			}
		}
	}
	return ""
}

func windowsNPMDirs() []string {
	var dirs []string
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" || containsStringFold(dirs, path) {
			return
		}
		dirs = append(dirs, path)
	}
	add(os.Getenv("NPM_CONFIG_PREFIX"))
	add(os.Getenv("npm_config_prefix"))
	if appData := os.Getenv("APPDATA"); appData != "" {
		add(filepath.Join(appData, "npm"))
	}
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		add(filepath.Join(userProfile, "AppData", "Roaming", "npm"))
	}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		add(filepath.Join(localAppData, "npm"))
	}
	add(os.Getenv("ProgramFiles") + `\nodejs`)
	add(os.Getenv("ProgramFiles(x86)") + `\nodejs`)
	return dirs
}

func ensurePathDir(env []string, dir string) []string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return env
	}
	for i, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok && strings.EqualFold(key, "PATH") {
			if pathListContains(value, dir) {
				return env
			}
			env[i] = key + "=" + dir + string(os.PathListSeparator) + value
			return env
		}
	}
	return append(env, "PATH="+dir)
}

func pathListContains(pathList, dir string) bool {
	for _, item := range filepath.SplitList(pathList) {
		if strings.EqualFold(filepath.Clean(item), filepath.Clean(dir)) {
			return true
		}
	}
	return false
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func containsStringFold(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(filepath.Clean(item), filepath.Clean(target)) {
			return true
		}
	}
	return false
}

func Truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "..."
}
