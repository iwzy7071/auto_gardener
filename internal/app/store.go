package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var ErrNotFound = errors.New("not found")

const (
	privateDirMode  os.FileMode = 0700
	privateFileMode os.FileMode = 0600
)

type taskDiskCompat struct {
	Task
	LegacyWave            int `json:"wave,omitempty"`
	LegacyMaxTreesPerWave int `json:"maxTreesPerWave,omitempty"`
}

type treeDiskCompat struct {
	Tree
	LegacyWave int `json:"wave,omitempty"`
}

type Store struct {
	mu       sync.RWMutex
	tasks    map[string]*Task
	dataDir  string
	events   *EventHub
	settings AppSettings
}

func NewStore(dataDir string, events *EventHub) (*Store, error) {
	if err := ensurePrivateDir(filepath.Join(dataDir, "forests")); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "workspaces"), 0755); err != nil {
		return nil, err
	}
	if err := ensurePrivateDir(filepath.Join(dataDir, "scratch")); err != nil {
		return nil, err
	}
	s := &Store{tasks: make(map[string]*Task), dataDir: dataDir, events: events, settings: defaultSettings()}
	if err := s.loadSettings(); err != nil {
		return nil, err
	}
	if err := s.Load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) DataDir() string { return s.dataDir }

func defaultSettings() AppSettings {
	return AppSettings{LogLevel: LogLevelQuiet, ModelMode: ModelModeDefault, CLIEngine: CLIEngineCodex}
}

func normalizeLogLevel(level LogLevel) LogLevel {
	switch level {
	case LogLevelQuiet, LogLevelNormal, LogLevelDetailed:
		return level
	default:
		return LogLevelQuiet
	}
}

func normalizeModelMode(mode ModelMode) ModelMode {
	switch mode {
	case ModelModeDefault, ModelModeMiniMax, ModelModeKimi:
		return mode
	default:
		return ModelModeDefault
	}
}

func normalizeCLIEngine(engine CLIEngine) CLIEngine {
	value := strings.ToLower(strings.TrimSpace(string(engine)))
	value = strings.ReplaceAll(value, "_", "-")
	switch value {
	case "codex", "openai-codex", "codex-cli", "openai":
		return CLIEngineCodex
	case "claude", "claude-code", "claude-cli", "anthropic", "cloud":
		// "cloud" is accepted as a common typo/alias for Claude in user data.
		return CLIEngineClaude
	default:
		return CLIEngineCodex
	}
}

func compatibleCLIEngine(engine CLIEngine, mode ModelMode) CLIEngine {
	engine = normalizeCLIEngine(engine)
	mode = normalizeModelMode(mode)
	if mode == ModelModeKimi && engine == CLIEngineCodex {
		// Kimi Coding rejects generic OpenAI/Codex-compatible requests and supports
		// coding agents such as Claude Code. Prefer a working, user-invisible path.
		return CLIEngineClaude
	}
	return engine
}

func (s *Store) settingsPath() string { return filepath.Join(s.dataDir, "settings.json") }

func (s *Store) loadSettings() error {
	var settings AppSettings
	if err := readJSON(s.settingsPath(), &settings); err != nil {
		if os.IsNotExist(err) {
			return s.persistSettingsLocked()
		}
		return err
	}
	settings.LogLevel = normalizeLogLevel(settings.LogLevel)
	settings.ModelMode = normalizeModelMode(settings.ModelMode)
	settings.CLIEngine = compatibleCLIEngine(settings.CLIEngine, settings.ModelMode)
	s.settings = settings
	return nil
}

func (s *Store) GetSettings() AppSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return normalizeSettings(s.settings)
}

func (s *Store) GetPublicSettings() AppSettings {
	return publicSettings(s.GetSettings())
}

func (s *Store) UpdateSettings(settings AppSettings) (AppSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings = normalizeSettings(settings)
	if strings.TrimSpace(settings.MiniMaxToken) == "" {
		settings.MiniMaxToken = s.settings.MiniMaxToken
	}
	if strings.TrimSpace(settings.KimiToken) == "" {
		settings.KimiToken = s.settings.KimiToken
	}
	s.settings = settings
	if err := s.persistSettingsLocked(); err != nil {
		return s.settings, err
	}
	return s.settings, nil
}

func normalizeSettings(settings AppSettings) AppSettings {
	settings.LogLevel = normalizeLogLevel(settings.LogLevel)
	settings.ModelMode = normalizeModelMode(settings.ModelMode)
	settings.CLIEngine = compatibleCLIEngine(settings.CLIEngine, settings.ModelMode)
	return settings
}

func (s *Store) persistSettingsLocked() error {
	path := s.settingsPath()
	if err := writeJSONFileMode(path, s.settings, 0600); err != nil {
		return err
	}
	return os.Chmod(path, 0600)
}

func (s *Store) Load() error {
	root := filepath.Join(s.dataDir, "forests")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		forestDir := filepath.Join(root, entry.Name())
		var diskTask taskDiskCompat
		if err := readJSON(filepath.Join(forestDir, "forest.json"), &diskTask); err != nil {
			continue
		}
		t := diskTask.Task
		if !safeDiskID(t.ID) || t.ID != entry.Name() {
			continue
		}
		normalizeForestFields(&t, diskTask.LegacyWave, diskTask.LegacyMaxTreesPerWave)
		_ = readJSON(filepath.Join(forestDir, "messages.json"), &t.Messages)
		if progress := readGardenerProgress(t.LogPath); len(progress) > 0 {
			t.GardenerProgress = progress
		}
		t.Trees = nil
		treesDir := filepath.Join(forestDir, "trees")
		if treeEntries, err := os.ReadDir(treesDir); err == nil {
			for _, te := range treeEntries {
				if !te.IsDir() {
					continue
				}
				var diskTree treeDiskCompat
				if err := readJSON(filepath.Join(treesDir, te.Name(), "tree.json"), &diskTree); err == nil {
					tr := diskTree.Tree
					if !safeDiskID(tr.ID) || tr.ID != te.Name() {
						continue
					}
					normalizeTreeForestFields(&tr, diskTree.LegacyWave)
					if tr.Progress == nil {
						tr.Progress = readProgress(filepath.Join(treesDir, te.Name(), "progress.log"))
					}
					t.Trees = append(t.Trees, &tr)
				}
			}
		}
		sort.Slice(t.Trees, func(i, j int) bool {
			if t.Trees[i].Forest == t.Trees[j].Forest {
				return t.Trees[i].CreatedOrder() < t.Trees[j].CreatedOrder()
			}
			return t.Trees[i].Forest < t.Trees[j].Forest
		})
		t.ModelMode = normalizeModelMode(t.ModelMode)
		t.CLIEngine = compatibleCLIEngine(t.CLIEngine, t.ModelMode)
		t.Status = normalizeStatus(t.Status)
		t.GardenerStatus = normalizeStatus(t.GardenerStatus)
		if strings.TrimSpace(t.ScratchPath) == "" {
			t.ScratchPath = t.WorkspacePath
		}
		if t.Status == StatusFinished && !t.StopRequested && hasLegacyInterruptedRun(t.LogPath) {
			t.Status = StatusRunning
			t.GardenerStatus = StatusRunning
		}
		for _, tr := range t.Trees {
			tr.Status = normalizeStatus(tr.Status)
			if tr.Status == StatusRunning {
				tr.Status = StatusFinished
				now := time.Now()
				tr.CompletedAt = &now
				progressPath := filepath.Join(forestDir, "trees", tr.ID, "progress.log")
				_ = appendFile(progressPath, fmt.Sprintf("[%s] 服务重启后发现该子任务的进程已不存在，已交由 Gardener 继续判断后续工作。\n", now.Format(time.RFC3339)))
			}
		}
		s.tasks[t.ID] = &t
		_ = s.persistTaskLocked(&t)
	}
	return nil
}

func safeDiskID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" || id == "." || id == ".." {
		return false
	}
	return id == filepath.Base(id) && !strings.ContainsAny(id, `/\`)
}

func normalizeForestFields(t *Task, legacyWave, legacyMaxTreesPerWave int) {
	if t.Forest == 0 && legacyWave > 0 {
		t.Forest = legacyWave
	}
	if t.MaxTreesPerForest == 0 && legacyMaxTreesPerWave > 0 {
		t.MaxTreesPerForest = legacyMaxTreesPerWave
	}
}

func normalizeTreeForestFields(tr *Tree, legacyWave int) {
	if tr.Forest == 0 && legacyWave > 0 {
		tr.Forest = legacyWave
	}
}

func hasLegacyInterruptedRun(logPath string) bool {
	data, err := os.ReadFile(logPath)
	if err != nil {
		return false
	}
	text := string(data)
	return strings.Contains(text, "服务重启后发现上次遗留 Running 状态，已恢复为 Finished")
}

func normalizeStatus(status Status) Status {
	if status == StatusRunning || status == StatusFinished {
		return status
	}
	return StatusFinished
}

func (tr *Tree) CreatedOrder() int64 {
	if tr.StartedAt != nil {
		return safeUnixNano(*tr.StartedAt)
	}
	return safeUnixNano(tr.UpdatedAt)
}

const (
	minInt64 = -1 << 63
	maxInt64 = 1<<63 - 1
)

var (
	minUnixNanoTime = time.Unix(0, minInt64)
	maxUnixNanoTime = time.Unix(0, maxInt64)
)

func safeUnixNano(t time.Time) int64 {
	if t.Before(minUnixNanoTime) {
		return minInt64
	}
	if t.After(maxUnixNanoTime) {
		return maxInt64
	}
	return t.UnixNano()
}

func (s *Store) AddTask(t *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	s.tasks[t.ID] = t
	if err := s.persistTaskLocked(t); err != nil {
		return err
	}
	s.publishLocked(t.ID)
	return nil
}

func (s *Store) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, cloneTask(t))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *Store) ListRunningTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Task, 0)
	for _, t := range s.tasks {
		if t.Status != StatusRunning {
			continue
		}
		out = append(out, cloneTask(t))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *Store) GetTask(id string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, false
	}
	return cloneTask(t), true
}

func (s *Store) DeleteTask(id string) error {
	s.mu.Lock()
	t, ok := s.tasks[id]
	if !ok {
		s.mu.Unlock()
		return ErrNotFound
	}
	task := cloneTask(t)
	delete(s.tasks, id)
	forestDir := s.forestDir(id)
	workspacePath := task.WorkspacePath
	scratchPath := task.ScratchPath
	s.mu.Unlock()

	if err := os.RemoveAll(forestDir); err != nil {
		return err
	}
	if shouldDeleteManagedWorkspace(s.dataDir, id, workspacePath) {
		if err := os.RemoveAll(workspacePath); err != nil {
			return err
		}
	}
	if shouldDeleteManagedScratch(s.dataDir, id, scratchPath) {
		if err := os.RemoveAll(scratchPath); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UpdateTask(id string, fn func(*Task)) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, ErrNotFound
	}
	fn(t)
	t.UpdatedAt = time.Now()
	if err := s.persistTaskLocked(t); err != nil {
		return nil, err
	}
	cp := cloneTask(t)
	s.publishLocked(id)
	return cp, nil
}

func (s *Store) AddTree(taskID string, tr *Tree) (*Task, error) {
	return s.UpdateTask(taskID, func(t *Task) {
		tr.UpdatedAt = time.Now()
		t.Trees = append(t.Trees, tr)
	})
}

func (s *Store) UpdateTree(taskID, treeID string, fn func(*Tree)) (*Task, error) {
	return s.UpdateTask(taskID, func(t *Task) {
		for _, tr := range t.Trees {
			if tr.ID == treeID {
				now := time.Now()
				fn(tr)
				tr.UpdatedAt = now
				t.LastProgressAt = &now
				return
			}
		}
	})
}

func (s *Store) FindTree(taskID, treeID string) (*Tree, error) {
	t, ok := s.GetTask(taskID)
	if !ok {
		return nil, ErrNotFound
	}
	for _, tr := range t.Trees {
		if tr.ID == treeID {
			return tr, nil
		}
	}
	return nil, ErrNotFound
}

func (s *Store) AppendTreeProgress(taskID, treeID, line string) {
	line = compactProgressLine(line)
	if line == "" {
		return
	}
	_, _ = s.UpdateTree(taskID, treeID, func(tr *Tree) {
		entry := fmt.Sprintf("%s %s", time.Now().Format("15:04:05"), line)
		if len(tr.Progress) > 0 && stripProgressTime(tr.Progress[len(tr.Progress)-1]) == line {
			return
		}
		tr.Progress = append(tr.Progress, entry)
		if len(tr.Progress) > 80 {
			tr.Progress = tr.Progress[len(tr.Progress)-80:]
		}
	})
	path := filepath.Join(s.dataDir, "forests", taskID, "trees", treeID, "progress.log")
	_ = appendFile(path, fmt.Sprintf("[%s] %s\n", time.Now().Format(time.RFC3339), line))
}

func (s *Store) AppendGardenerLog(taskID, line string) {
	line = compactProgressLine(line)
	if line == "" {
		return
	}
	now := time.Now()
	entry := fmt.Sprintf("%s %s", now.Format("15:04:05"), line)

	s.mu.Lock()
	t, ok := s.tasks[taskID]
	if !ok {
		s.mu.Unlock()
		return
	}
	logPath := t.LogPath
	if len(t.GardenerProgress) == 0 || stripProgressTime(t.GardenerProgress[len(t.GardenerProgress)-1]) != line {
		t.GardenerProgress = append(t.GardenerProgress, entry)
		if len(t.GardenerProgress) > 80 {
			t.GardenerProgress = t.GardenerProgress[len(t.GardenerProgress)-80:]
		}
	}
	t.LastProgressAt = &now
	t.UpdatedAt = now
	cp := cloneTask(t)
	s.mu.Unlock()

	_ = appendFile(logPath, fmt.Sprintf("\n[%s] %s\n", now.Format(time.RFC3339), line))
	s.events.Publish(taskID, cp)
}

func (s *Store) WriteSchedule(taskID, content string) error {
	t, ok := s.GetTask(taskID)
	if !ok {
		return ErrNotFound
	}
	if err := os.MkdirAll(filepath.Dir(t.SchedulePath), 0755); err != nil {
		return err
	}
	if err := writePrivateFile(t.SchedulePath, []byte(content)); err != nil {
		return err
	}
	s.events.Publish(taskID, t)
	return nil
}

func (s *Store) AppendSchedule(taskID, content string) error {
	t, ok := s.GetTask(taskID)
	if !ok {
		return ErrNotFound
	}
	if err := appendScheduleFile(t.SchedulePath, content); err != nil {
		return err
	}
	s.events.Publish(taskID, t)
	return nil
}

func (s *Store) publishLocked(taskID string) {
	if s.events == nil {
		return
	}
	if t, ok := s.tasks[taskID]; ok {
		s.events.Publish(taskID, cloneTask(t))
	}
}

func (s *Store) forestDir(taskID string) string {
	return filepath.Join(s.dataDir, "forests", taskID)
}

func (s *Store) persistTaskLocked(t *Task) error {
	forestDir := s.forestDir(t.ID)
	if err := ensurePrivateDir(filepath.Join(forestDir, "gardener")); err != nil {
		return err
	}
	if err := ensurePrivateDir(filepath.Join(forestDir, "trees")); err != nil {
		return err
	}
	meta := cloneTask(t)
	meta.Messages = nil
	meta.Trees = nil
	meta.Runtime = nil
	if err := writePrivateJSONFile(filepath.Join(forestDir, "forest.json"), meta); err != nil {
		return err
	}
	if err := writePrivateJSONFile(filepath.Join(forestDir, "messages.json"), t.Messages); err != nil {
		return err
	}
	for _, tr := range t.Trees {
		dir := filepath.Join(forestDir, "trees", tr.ID)
		if err := ensurePrivateDir(dir); err != nil {
			return err
		}
		if err := writePrivateJSONFile(filepath.Join(dir, "tree.json"), tr); err != nil {
			return err
		}
	}
	return nil
}

func cloneTask(t *Task) *Task {
	cp := *t
	cp.Runtime = buildTaskRuntime(t, time.Now())
	cp.Messages = append([]Message(nil), t.Messages...)
	cp.GardenerProgress = append([]string(nil), t.GardenerProgress...)
	cp.Trees = make([]*Tree, 0, len(t.Trees))
	for _, tr := range t.Trees {
		cp.Trees = append(cp.Trees, cloneTree(tr))
	}
	return &cp
}

func cloneTree(tr *Tree) *Tree {
	cp := *tr
	cp.Scope = append([]string(nil), tr.Scope...)
	cp.Progress = append([]string(nil), tr.Progress...)
	return &cp
}

func writeRecoveryFruit(path string, task *Task, tr *Tree, when time.Time) error {
	body := fmt.Sprintf(`# 子任务工作报告

## 1. 子任务基本信息

- 子任务 ID: %s
- 所属任务 ID: %s
- 所属任务: %s
- 子任务名称: %s
- 状态: Finished
- 结束时间: %s
- 目标项目目录: %s

## 2. 子任务目标

%s

## 3. 执行过程

服务重启后发现该子任务在上次进程中遗留为 Running。由于原底层 CLI 进程已不存在，系统将其恢复为 Finished。

## 4. 完成结果

未能确认该子任务上次运行的完整完成结果。请查看 progress.log、workspace git diff 和 Gardener log.md。

## 5. 产出文件或关键修改

请查看 workspace 当前文件状态和 git diff。

## 6. 遇到的问题

服务重启恢复产生的报告。

## 7. 对 Gardener 的汇报

该子任务是从遗留 Running 状态恢复为 Finished。

## 8. 后续建议

建议由 Gardener 派验证子任务或修复子任务检查 workspace 状态。
`, tr.ID, task.ID, task.Title, tr.Name, when.Format(time.RFC3339), task.WorkspacePath, tr.Objective)
	return writePrivateFile(path, []byte(body))
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func writeJSONFile(path string, v any) error {
	return writeJSONFileMode(path, v, 0644)
}

func writeJSONFileMode(path string, v any, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, perm)
}

func writePrivateJSONFile(path string, v any) error {
	if err := ensurePrivateDir(filepath.Dir(path)); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writePrivateFile(path, b)
}

func ensurePrivateDir(path string) error {
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to use symlink directory: %s", path)
	}
	if err := os.MkdirAll(path, privateDirMode); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to use symlink directory: %s", path)
	}
	return os.Chmod(path, privateDirMode)
}

func writePrivateFile(path string, b []byte) error {
	if err := os.WriteFile(path, b, privateFileMode); err != nil {
		return err
	}
	return os.Chmod(path, privateFileMode)
}

func compactProgressLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	line = strings.Join(strings.Fields(line), " ")
	runes := []rune(line)
	if len(runes) > 500 {
		line = string(runes[:500]) + "..."
	}
	return line
}

func stripProgressTime(line string) string {
	if len(line) > 9 && line[2] == ':' && line[5] == ':' && line[8] == ' ' {
		return line[9:]
	}
	return line
}

func shouldDeleteManagedWorkspace(dataDir, taskID, workspacePath string) bool {
	if strings.TrimSpace(workspacePath) == "" {
		return false
	}
	root, err := filepath.Abs(filepath.Join(dataDir, "workspaces"))
	if err != nil {
		return false
	}
	workspace, err := filepath.Abs(filepath.Clean(workspacePath))
	if err != nil {
		return false
	}
	if workspace == root {
		return false
	}
	rel, err := filepath.Rel(root, workspace)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return false
	}
	base := filepath.Base(workspace)
	return base == taskID || strings.HasPrefix(base, taskID+"_")
}

func shouldDeleteManagedScratch(dataDir, taskID, scratchPath string) bool {
	if strings.TrimSpace(scratchPath) == "" {
		return false
	}
	roots := []string{
		filepath.Join(os.TempDir(), "GardenerScratch"),
		filepath.Join(dataDir, "scratch"),
	}
	scratch, err := filepath.Abs(filepath.Clean(scratchPath))
	if err != nil {
		return false
	}
	for _, rootPath := range roots {
		root, err := filepath.Abs(rootPath)
		if err != nil || scratch == root {
			continue
		}
		rel, err := filepath.Rel(root, scratch)
		if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			continue
		}
		base := filepath.Base(scratch)
		return base == taskID || strings.HasPrefix(base, taskID+"_")
	}
	return false
}

func appendFile(path, s string) error {
	if err := ensurePrivateDir(filepath.Dir(path)); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, privateFileMode)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Chmod(privateFileMode); err != nil {
		return err
	}
	_, err = f.WriteString(s)
	return err
}

func appendScheduleFile(path, s string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() > 0 {
		var last [1]byte
		if _, err := f.ReadAt(last[:], info.Size()-1); err != nil {
			return err
		}
		if last[0] != '\n' {
			if _, err := f.WriteString("\n"); err != nil {
				return err
			}
		}
	}
	_, err = f.WriteString(s)
	return err
}

func readProgress(path string) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := stringsSplitLines(string(b))
	if len(lines) > 80 {
		return lines[len(lines)-80:]
	}
	return lines
}

func readGardenerProgress(path string) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, raw := range stringsSplitLines(string(b)) {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") {
			if end := strings.Index(line, "]"); end > 1 {
				stamp := line[1:end]
				body := strings.TrimSpace(line[end+1:])
				if ts, err := time.Parse(time.RFC3339, stamp); err == nil {
					line = fmt.Sprintf("%s %s", ts.Format("15:04:05"), body)
				} else {
					line = body
				}
			}
		}
		line = compactProgressLine(line)
		if line != "" {
			out = append(out, line)
		}
	}
	if len(out) > 80 {
		return out[len(out)-80:]
	}
	return out
}

func stringsSplitLines(s string) []string {
	var out []string
	start := 0
	for i, r := range s {
		if r == '\n' {
			line := s[start:i]
			if line != "" {
				out = append(out, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
