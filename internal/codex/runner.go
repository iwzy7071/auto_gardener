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
			line := s.Text()
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
	output := out.String()
	mu.Unlock()
	if req.OutputFile != "" {
		if b, readErr := os.ReadFile(req.OutputFile); readErr == nil && len(b) > 0 {
			// The Codex final message is the authoritative output. stdout/stderr is
			// already streamed through OnLine into log.md/progress.log. Keeping only
			// the final message here prevents Gardener JSON parsing from being polluted
			// by CLI progress text.
			output = string(b)
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
	args := []string{
		"-p",
		"--output-format", "text",
		"--permission-mode", "bypassPermissions",
	}
	if model := strings.TrimSpace(os.Getenv("AUTO_GARDENER_CLAUDE_MODEL")); model != "" {
		args = append(args, "--model", model)
	}
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = req.WorkDir
	cmd.Env = appendClaudeEnv(env, req.Model)
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
			line := s.Text()
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
	output := out.String()
	mu.Unlock()
	if req.OutputFile != "" && strings.TrimSpace(output) != "" {
		_ = os.MkdirAll(filepath.Dir(req.OutputFile), 0755)
		_ = os.WriteFile(req.OutputFile, []byte(output), 0644)
	}
	return RunResult{Output: output, Err: err}
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
	if strings.TrimSpace(model.ProviderID) == "gardener-kimi" && token != "" {
		env = upsertEnv(env, "ANTHROPIC_BASE_URL", firstNonEmpty(os.Getenv("AUTO_GARDENER_KIMI_CLAUDE_BASE_URL"), "https://api.kimi.com/coding/"))
		env = upsertEnv(env, "ANTHROPIC_API_KEY", token)
		env = upsertEnv(env, "ENABLE_TOOL_SEARCH", firstNonEmpty(os.Getenv("AUTO_GARDENER_KIMI_ENABLE_TOOL_SEARCH"), "false"))
	}
	return env
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
		return "", env, fmt.Errorf("找不到 Codex CLI 命令 %q", command)
	}
	if runtime.GOOS == "windows" {
		if path := findWindowsNPMCommand(command); path != "" {
			env = ensurePathDir(env, filepath.Dir(path))
			return path, env, nil
		}
	}
	return "", env, fmt.Errorf("找不到 %s 命令 %q；如果 CLI 是用 npm 在 Windows 安装，请确认 %%APPDATA%%\\npm 下存在对应 .cmd 文件，或设置 %s", label, command, envVar)
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
