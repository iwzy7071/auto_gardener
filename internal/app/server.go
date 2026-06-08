package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type workspaceFileEntry struct {
	Path    string   `json:"path"`
	Size    int64    `json:"size"`
	ModTime string   `json:"modTime"`
	TreeIDs []string `json:"treeIds,omitempty"`
}

type Server struct {
	store        *Store
	orchestrator *Orchestrator
	staticDir    string
	events       *EventHub

	dingTalkMu           sync.Mutex
	dingTalkSessions     map[string]string
	dingTalkSessionOrder []string
	httpClient           *http.Client
}

func NewServer(store *Store, orchestrator *Orchestrator, staticDir string, events *EventHub) *Server {
	return &Server{store: store, orchestrator: orchestrator, staticDir: staticDir, events: events, dingTalkSessions: make(map[string]string), httpClient: newDingTalkHTTPClient()}
}

func newDingTalkHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/tasks", s.handleTasks)
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/usage", s.handleUsage)
	mux.HandleFunc("/api/fs/dirs", s.handleDirectoryBrowse)
	mux.HandleFunc("/api/dingtalk/robot", s.handleDingTalkRobot)
	mux.HandleFunc("/api/tasks/", s.handleTaskSubroutes)
	mux.HandleFunc("/", s.serveStaticApp)
	return logRequests(rejectCrossOriginAPIWrites(mux))
}

func rejectCrossOriginAPIWrites(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isUnsafeMethod(r.Method) && strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/dingtalk/robot" {
			if !requestHasSameOrigin(r) {
				writeError(w, http.StatusForbidden, "跨站请求已被拒绝")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func requestHasSameOrigin(r *http.Request) bool {
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		return requestOriginAllowed(r, origin)
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		return requestOriginAllowed(r, referer)
	}
	return true
}

func requestOriginAllowed(r *http.Request, rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return false
	}
	for _, allowed := range allowedOriginHosts(r) {
		if hostsMatchForCSRF(u.Host, allowed) {
			return true
		}
	}
	for _, allowed := range configuredAllowedOrigins() {
		if originMatchesConfiguredAllowed(u, allowed) {
			return true
		}
	}
	return false
}

func allowedOriginHosts(r *http.Request) []string {
	var hosts []string
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				hosts = append(hosts, part)
			}
		}
	}
	add(r.Host)
	add(r.Header.Get("X-Forwarded-Host"))
	add(r.Header.Get("X-Original-Host"))
	if forwarded := strings.TrimSpace(r.Header.Get("Forwarded")); forwarded != "" {
		for _, item := range strings.Split(forwarded, ";") {
			key, value, ok := strings.Cut(strings.TrimSpace(item), "=")
			if ok && strings.EqualFold(strings.TrimSpace(key), "host") {
				add(strings.Trim(strings.TrimSpace(value), `"`))
			}
		}
	}
	return hosts
}

func configuredAllowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("AUTO_GARDENER_ALLOWED_ORIGINS"))
	if raw == "" {
		return nil
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == '\n' || r == '\t' || r == ' ' })
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}

func originMatchesConfiguredAllowed(origin *url.URL, allowed string) bool {
	if allowed == "*" {
		return true
	}
	if strings.Contains(allowed, "://") {
		u, err := url.Parse(allowed)
		if err != nil || u.Host == "" {
			return false
		}
		return strings.EqualFold(origin.Scheme, u.Scheme) && hostsMatchForCSRF(origin.Host, u.Host)
	}
	return hostsMatchForCSRF(origin.Host, allowed)
}

func hostsMatchForCSRF(originHost, requestHost string) bool {
	originHost = strings.TrimSpace(originHost)
	requestHost = strings.TrimSpace(requestHost)
	if originHost == "" || requestHost == "" {
		return false
	}
	if strings.EqualFold(originHost, requestHost) {
		return true
	}
	originName, originPort := splitHostPortLoose(originHost)
	requestName, requestPort := splitHostPortLoose(requestHost)
	if originName == "" || requestName == "" || !strings.EqualFold(originName, requestName) {
		return false
	}
	// Some reverse proxies pass Host as $host, which strips the public port.
	// Treat same-host requests as same-origin when one side has no explicit port,
	// while still rejecting different explicit ports.
	return originPort == "" || requestPort == ""
}

func splitHostPortLoose(host string) (string, string) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", ""
	}
	u, err := url.Parse("//" + host)
	if err == nil && u.Hostname() != "" {
		return strings.ToLower(u.Hostname()), u.Port()
	}
	if h, p, ok := strings.Cut(host, ":"); ok && !strings.Contains(p, ":") {
		return strings.ToLower(strings.Trim(h, "[]")), p
	}
	return strings.ToLower(strings.Trim(host, "[]")), ""
}

func headerURLMatchesHost(rawURL, host string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return false
	}
	return hostsMatchForCSRF(u.Host, host)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"version": Version,
		"time":    time.Now().Format(time.RFC3339),
		"power":   CheckPowerStatus(),
	})
}

func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"usage": s.store.AllUsage()})
}

func (s *Server) serveStaticApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	staticRoot, err := filepath.Abs(filepath.Clean(s.staticDir))
	if err != nil || staticRoot == "" {
		http.NotFound(w, r)
		return
	}
	path := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if path == "." || path == "/" {
		http.ServeFile(w, r, filepath.Join(staticRoot, "index.html"))
		return
	}
	abs, err := filepath.Abs(filepath.Join(staticRoot, path))
	if err != nil || (abs != staticRoot && !strings.HasPrefix(abs, staticRoot+string(filepath.Separator))) {
		http.NotFound(w, r)
		return
	}
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		http.ServeFile(w, r, abs)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/forests/") {
		http.ServeFile(w, r, filepath.Join(staticRoot, "index.html"))
		return
	}
	http.NotFound(w, r)
}

func compactTaskList(tasks []*Task) []*Task {
	out := make([]*Task, 0, len(tasks))
	for _, task := range tasks {
		if task == nil {
			continue
		}
		cp := *task
		cp.Prompt = ""
		cp.WorkspacePath = ""
		cp.ScratchPath = ""
		cp.SchedulePath = ""
		cp.LogPath = ""
		cp.Messages = nil
		cp.GardenerProgress = nil
		cp.Trees = make([]*Tree, 0, len(task.Trees))
		for _, tr := range task.Trees {
			if tr == nil {
				continue
			}
			fruitReady := ""
			if tr.FruitPath != "" {
				fruitReady = "ready"
			}
			cp.Trees = append(cp.Trees, &Tree{
				ID:           tr.ID,
				TaskID:       tr.TaskID,
				Forest:       tr.Forest,
				Name:         tr.Name,
				IsValidation: tr.IsValidation,
				Status:       tr.Status,
				FruitPath:    fruitReady,
				UpdatedAt:    tr.UpdatedAt,
			})
		}
		out = append(out, &cp)
	}
	return out
}

func publicTasks(tasks []*Task) []*Task {
	out := make([]*Task, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, publicTask(task))
	}
	return out
}

func publicTask(task *Task) *Task {
	if task == nil {
		return nil
	}
	cp := *task
	cp.Prompt = ""
	cp.WorkspacePath = ""
	cp.ScratchPath = ""
	cp.SchedulePath = ""
	cp.LogPath = ""
	cp.Trees = make([]*Tree, 0, len(task.Trees))
	for _, tr := range task.Trees {
		if tr == nil {
			continue
		}
		tc := *tr
		tc.FruitPath = publicReadyMarker(tr.FruitPath)
		tc.GoalPath = ""
		cp.Trees = append(cp.Trees, &tc)
	}
	return &cp
}

func publicReadyMarker(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return "ready"
}

type directoryEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (s *Server) handleDirectoryBrowse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	requested := strings.TrimSpace(r.URL.Query().Get("path"))
	current := requested
	if current == "" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			current = home
		} else {
			current = string(filepath.Separator)
		}
	}
	current = expandHome(current)
	abs, err := filepath.Abs(filepath.Clean(current))
	if err != nil {
		writeError(w, http.StatusBadRequest, "目录路径无效")
		return
	}
	if !isAllowedDirectoryBrowsePath(abs) {
		writeError(w, http.StatusForbidden, "只能浏览用户主目录内的目录")
		return
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		writeError(w, http.StatusBadRequest, "目录不存在或不可访问")
		return
	}
	entries, _ := os.ReadDir(abs)
	dirs := make([]directoryEntry, 0)
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		path := filepath.Join(abs, entry.Name())
		if _, err := os.Stat(path); err != nil {
			continue
		}
		dirs = append(dirs, directoryEntry{Name: entry.Name(), Path: path})
	}
	sort.Slice(dirs, func(i, j int) bool { return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name) })
	parent := filepath.Dir(abs)
	if parent == abs || !isAllowedDirectoryBrowsePath(parent) {
		parent = ""
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":    abs,
		"parent":  parent,
		"entries": dirs,
	})
}

func isAllowedDirectoryBrowsePath(path string) bool {
	if os.Getenv("AUTO_GARDENER_ALLOW_DIRECTORY_BROWSE_ROOT") == "1" {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return false
	}
	root, err := filepath.Abs(filepath.Clean(home))
	if err != nil {
		return false
	}
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return false
	}
	return abs == root || strings.HasPrefix(abs, root+string(filepath.Separator))
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tasks := s.store.ListTasks()
		if r.URL.Query().Get("compact") == "1" {
			tasks = compactTaskList(tasks)
		}
		writeJSON(w, http.StatusOK, map[string]any{"tasks": publicTasks(tasks)})
	case http.MethodPost:
		var req CreateTaskRequest
		if !decodeLimitedJSON(w, r, &req, maxTaskJSONBodyBytes, "请求体不是合法 JSON") {
			return
		}
		task, err := s.orchestrator.CreateTask(req.Prompt, req.WorkspacePath)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, CreateTaskResponse{Task: publicTask(task)})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, SettingsResponse{Settings: s.store.GetPublicSettings()})
	case http.MethodPut:
		var settings AppSettings
		if !decodeLimitedJSON(w, r, &settings, maxSettingsJSONBodyBytes, "请求体不是合法 JSON") {
			return
		}
		updated, err := s.store.UpdateSettings(settings)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, SettingsResponse{Settings: publicSettings(updated)})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func publicSettings(settings AppSettings) AppSettings {
	settings = normalizeSettings(settings)
	settings.MiniMaxToken = ""
	settings.KimiToken = ""
	return settings
}

func (s *Server) handleTaskSubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	taskID := parts[0]
	if len(parts) == 1 {
		if r.Method != http.MethodGet && r.Method != http.MethodDelete && r.Method != http.MethodPatch {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Method == http.MethodGet {
			task, ok := s.store.GetTask(taskID)
			if !ok {
				writeError(w, http.StatusNotFound, "任务不存在")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"task": publicTask(task)})
			return
		}
		if r.Method == http.MethodPatch {
			var req RenameTaskRequest
			if !decodeLimitedJSON(w, r, &req, maxMessageJSONBodyBytes, "请求体不是合法 JSON") {
				return
			}
			task, err := s.orchestrator.RenameTask(taskID, req.Title)
			if err != nil {
				status := http.StatusBadRequest
				if errors.Is(err, ErrNotFound) {
					status = http.StatusNotFound
				}
				writeError(w, status, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"task": publicTask(task)})
			return
		}
		if err := s.orchestrator.DeleteTask(taskID); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, ErrNotFound) {
				status = http.StatusNotFound
			}
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
		return
	}

	if len(parts) == 2 && parts[1] == "events" && r.Method == http.MethodGet {
		s.handleEvents(w, r, taskID)
		return
	}

	if len(parts) == 2 && parts[1] == "diagnostics" && r.Method == http.MethodGet {
		task, ok := s.store.GetTask(taskID)
		if !ok {
			writeError(w, http.StatusNotFound, "任务不存在")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"runtime": task.Runtime})
		return
	}

	if len(parts) == 2 && parts[1] == "usage" && r.Method == http.MethodGet {
		usage, err := s.store.TaskUsage(taskID)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, ErrNotFound) {
				status = http.StatusNotFound
			}
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"usage": usage})
		return
	}

	if len(parts) == 2 && parts[1] == "messages" && r.Method == http.MethodPost {
		var req SendMessageRequest
		if !decodeLimitedJSON(w, r, &req, maxMessageJSONBodyBytes, "请求体不是合法 JSON") {
			return
		}
		task, err := s.orchestrator.SendMessage(taskID, req.Content)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, ErrNotFound) {
				status = http.StatusNotFound
			}
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"task": publicTask(task)})
		return
	}

	if len(parts) == 2 && parts[1] == "stop" && r.Method == http.MethodPost {
		task, err := s.orchestrator.StopTask(taskID)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, ErrNotFound) {
				status = http.StatusNotFound
			}
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"task": publicTask(task)})
		return
	}

	if len(parts) == 2 && parts[1] == "resume" && r.Method == http.MethodPost {
		task, err := s.orchestrator.ResumeTask(taskID)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, ErrNotFound) {
				status = http.StatusNotFound
			}
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"task": publicTask(task)})
		return
	}

	if len(parts) == 2 && parts[1] == "files" && r.Method == http.MethodGet {
		task, ok := s.store.GetTask(taskID)
		if !ok {
			writeError(w, http.StatusNotFound, "任务不存在")
			return
		}
		if rel := strings.TrimSpace(r.URL.Query().Get("path")); rel != "" {
			s.serveWorkspaceFile(w, r, task, rel)
			return
		}
		s.listWorkspaceFiles(w, r, task)
		return
	}

	if len(parts) == 3 && parts[1] == "gardener" && r.Method == http.MethodGet {
		task, ok := s.store.GetTask(taskID)
		if !ok {
			writeError(w, http.StatusNotFound, "任务不存在")
			return
		}
		switch parts[2] {
		case "schedule.md":
			s.serveMarkdown(w, r, task.SchedulePath)
		case "log.md":
			s.serveMarkdown(w, r, task.LogPath)
		default:
			writeError(w, http.StatusNotFound, "not found")
		}
		return
	}

	if len(parts) == 3 && parts[1] == "trees" && r.Method == http.MethodGet {
		tree, err := s.store.FindTree(taskID, parts[2])
		if err != nil {
			writeError(w, http.StatusNotFound, "Tree 不存在")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"tree": tree})
		return
	}

	if len(parts) == 4 && parts[1] == "trees" && parts[3] == "fruit.md" && r.Method == http.MethodGet {
		tree, err := s.store.FindTree(taskID, parts[2])
		if err != nil || tree.FruitPath == "" {
			writeError(w, http.StatusNotFound, "fruit.md 尚不存在")
			return
		}
		s.serveMarkdown(w, r, tree.FruitPath)
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request, taskID string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "当前响应不支持 SSE")
		return
	}
	if _, ok := s.store.GetTask(taskID); !ok {
		writeError(w, http.StatusNotFound, "任务不存在")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch, unsubscribe := s.events.Subscribe(taskID)
	defer unsubscribe()
	if task, ok := s.store.GetTask(taskID); ok {
		writeSSE(w, "task", publicTask(task))
		flusher.Flush()
	}
	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()
	flushTicker := time.NewTicker(750 * time.Millisecond)
	defer flushTicker.Stop()
	var pending *Task
	lastFlush := time.Now()
	flushPending := func(force bool) {
		if pending == nil {
			return
		}
		if !force && time.Since(lastFlush) < 650*time.Millisecond {
			return
		}
		writeSSE(w, "task", publicTask(pending))
		flusher.Flush()
		pending = nil
		lastFlush = time.Now()
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case task, ok := <-ch:
			if !ok {
				return
			}
			pending = task
			if task.Status == StatusFinished || task.GardenerStatus == StatusFinished {
				flushPending(true)
			} else {
				flushPending(false)
			}
		case <-flushTicker.C:
			flushPending(true)
		case <-keepalive.C:
			flushPending(true)
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, event string, v any) {
	b, _ := json.Marshal(v)
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", b)
}

func (s *Server) listWorkspaceFiles(w http.ResponseWriter, r *http.Request, task *Task) {
	root, err := filepath.Abs(filepath.Clean(task.WorkspacePath))
	if err != nil || root == "" {
		writeError(w, http.StatusBadRequest, "保存位置无效")
		return
	}
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		writeJSON(w, http.StatusOK, map[string]any{"files": []workspaceFileEntry{}})
		return
	}
	filterTree := strings.TrimSpace(r.URL.Query().Get("treeId"))
	filterForest := strings.TrimSpace(r.URL.Query().Get("forest"))
	forestTreeIDs := treeIDsForForest(task, filterForest)
	matches := treeScopeMatchers(task)
	files := make([]workspaceFileEntry, 0)
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if name == ".git" || name == "node_modules" || name == ".next" || name == "dist" || name == "build" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if strings.TrimSpace(task.ScratchPath) != "" && filepath.Clean(task.ScratchPath) != filepath.Clean(task.WorkspacePath) {
			if info.ModTime().Before(task.CreatedAt.Add(-2 * time.Minute)) {
				return nil
			}
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if isHiddenOrNoiseFile(rel) || isSensitiveWorkspaceFile(rel) {
			return nil
		}
		treeIDs := matchingTreeIDs(rel, matches)
		if filterTree != "" && !containsString(treeIDs, filterTree) {
			return nil
		}
		if filterTree == "" && len(forestTreeIDs) > 0 && !intersectsString(treeIDs, forestTreeIDs) {
			return nil
		}
		files = append(files, workspaceFileEntry{Path: rel, Size: info.Size(), ModTime: info.ModTime().Format(time.RFC3339), TreeIDs: treeIDs})
		if len(files) >= 500 {
			return filepath.SkipAll
		}
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

func (s *Server) serveWorkspaceFile(w http.ResponseWriter, r *http.Request, task *Task, rel string) {
	root, err := filepath.Abs(filepath.Clean(task.WorkspacePath))
	if err != nil || root == "" {
		writeError(w, http.StatusBadRequest, "保存位置无效")
		return
	}
	rel = filepath.Clean(strings.TrimPrefix(rel, "/"))
	if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		writeError(w, http.StatusBadRequest, "文件路径非法")
		return
	}
	if isSensitiveWorkspaceFile(rel) {
		writeError(w, http.StatusForbidden, "敏感文件不允许通过工作区预览或下载")
		return
	}
	path := filepath.Join(root, rel)
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil || (abs != root && !strings.HasPrefix(abs, root+string(filepath.Separator))) {
		writeError(w, http.StatusForbidden, "只能读取保存位置内的文件")
		return
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "文件不存在")
		return
	}
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", contentDisposition("attachment", abs))
		http.ServeFile(w, r, abs)
		return
	}
	if info.Size() > 2*1024*1024 {
		writeError(w, http.StatusBadRequest, "文件过大，暂不支持预览")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	http.ServeFile(w, r, abs)
}

func treeIDsForForest(task *Task, forestFilter string) []string {
	if forestFilter == "" {
		return nil
	}
	forest, err := strconv.Atoi(forestFilter)
	if err != nil || forest <= 0 {
		return nil
	}
	ids := make([]string, 0)
	for _, tr := range task.Trees {
		if tr.Forest == forest && !tr.IsValidation {
			ids = append(ids, tr.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func intersectsString(items []string, targets []string) bool {
	for _, item := range items {
		for _, target := range targets {
			if item == target {
				return true
			}
		}
	}
	return false
}

func treeScopeMatchers(task *Task) map[string][]string {
	out := make(map[string][]string)
	for _, tr := range task.Trees {
		if tr.IsValidation {
			continue
		}
		for _, scope := range tr.Scope {
			scope = filepath.ToSlash(strings.TrimSpace(scope))
			scope = strings.Trim(scope, "/")
			if scope != "" && scope != "." {
				out[tr.ID] = append(out[tr.ID], scope)
			}
		}
	}
	return out
}

func matchingTreeIDs(rel string, matchers map[string][]string) []string {
	var ids []string
	for id, scopes := range matchers {
		for _, scope := range scopes {
			if rel == scope || strings.HasPrefix(rel, strings.TrimSuffix(scope, "/")+"/") || strings.HasPrefix(scope, rel+"/") {
				ids = append(ids, id)
				break
			}
		}
	}
	sort.Strings(ids)
	return ids
}

func isHiddenOrNoiseFile(rel string) bool {
	base := filepath.Base(rel)
	if strings.HasPrefix(base, ".") {
		return true
	}
	lower := strings.ToLower(base)
	if lower == "package-lock.json" || lower == "pnpm-lock.yaml" || lower == "yarn.lock" {
		return true
	}
	return strings.HasSuffix(lower, ".tmp") || strings.HasSuffix(lower, ".temp") || strings.HasSuffix(lower, ".part") || strings.HasSuffix(lower, ".crdownload") || strings.HasSuffix(lower, ".log") || strings.HasSuffix(lower, ".bak")
}

func isSensitiveWorkspaceFile(rel string) bool {
	base := strings.ToLower(filepath.Base(rel))
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}
	switch base {
	case ".npmrc", ".pypirc", ".netrc", "id_rsa", "id_dsa", "id_ecdsa", "id_ed25519":
		return true
	}
	return strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".key")
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func (s *Server) serveMarkdown(w http.ResponseWriter, r *http.Request, path string) {
	if path == "" {
		writeError(w, http.StatusNotFound, "文件不存在")
		return
	}
	cleanPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		writeError(w, http.StatusBadRequest, "文件路径非法")
		return
	}
	dataRoot, err := filepath.Abs(filepath.Clean(s.store.DataDir()))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "数据目录非法")
		return
	}
	realRoot, err := filepath.EvalSymlinks(dataRoot)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "数据目录不存在或不可访问")
		return
	}
	realPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "文件不存在")
		return
	}
	if realPath != realRoot && !strings.HasPrefix(realPath, realRoot+string(filepath.Separator)) {
		writeError(w, http.StatusForbidden, "只能读取 forest_data 内的报告文件")
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", contentDisposition("inline", realPath))
	http.ServeFile(w, r, realPath)
}

func contentDisposition(disposition, path string) string {
	return mime.FormatMediaType(disposition, map[string]string{"filename": filepath.Base(path)})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && !strings.HasSuffix(r.URL.Path, "/events") {
			log.Printf("%s %s", r.Method, r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}
