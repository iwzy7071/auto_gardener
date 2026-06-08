package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"auto_gardener/internal/codex"
)

type Orchestrator struct {
	store         *Store
	runner        codex.Runner
	dataDir       string
	maxTrees      int
	maxConcurrent int
	compatBaseURL string

	mu         sync.Mutex
	cancels    map[string]context.CancelFunc
	activeRuns map[string]string
}

func NewOrchestrator(store *Store, runner codex.Runner, dataDir, compatBaseURL string) *Orchestrator {
	return &Orchestrator{
		store:         store,
		runner:        runner,
		dataDir:       dataDir,
		maxTrees:      getenvIntFallback("AUTO_GARDENER_MAX_TREES_PER_FOREST", "AUTO_GARDENER_MAX_TREES_PER_WAVE", 5),
		maxConcurrent: getenvInt("AUTO_GARDENER_MAX_CONCURRENT_TREES", 3),
		compatBaseURL: strings.TrimRight(compatBaseURL, "/"),
		cancels:       make(map[string]context.CancelFunc),
		activeRuns:    make(map[string]string),
	}
}

func (o *Orchestrator) ResumeUnfinished() {
	for _, task := range o.store.ListTasks() {
		if task.Status != StatusRunning || task.StopRequested {
			continue
		}
		_, _ = o.store.UpdateTask(task.ID, func(t *Task) {
			t.Status = StatusRunning
			t.GardenerStatus = StatusRunning
		})
		o.store.AppendGardenerLog(task.ID, "服务启动后继续未完成的任务。")
		o.startForestRun(task.ID, buildResumeInstruction(task))
	}
}

func buildResumeInstruction(t *Task) string {
	return fmt.Sprintf(`服务刚刚启动，请继续这个尚未完成的任务。

要求：
- 不要重复已经足够完成的工作。
- 先检查交付目录、schedule、已有子任务报告和当前文件。
- 如果已有成果足够，直接给出 message_to_user 并结束。
- 如果仍缺少内容，请继续派出新的子任务完成剩余工作。

任务 ID: %s
任务标题: %s
outputPath: %s
scratchPath: %s
当前阶段: %d
`, t.ID, t.Title, t.WorkspacePath, taskWorkDir(t), t.Forest)
}

func buildContinueInstruction(t *Task) string {
	return fmt.Sprintf(`用户点击了“继续任务”。请从当前进度继续，而不是从头重做。

要求：
- 先检查交付目录、schedule、已有子任务报告、验证报告、progress.log 和当前文件状态。
- 如果任务已经足够完成，请向用户说明已完成，并结束。
- 如果仍有缺口、失败、冲突、半成品或未交付内容，请继续派出新的子任务完成剩余工作。
- 如果上一次是因为 CLI 异常、JSON 解析失败或用户手动停止而暂停，请优先定位未完成部分并继续。

任务 ID: %s
任务标题: %s
outputPath: %s
scratchPath: %s
当前阶段: %d
`, t.ID, t.Title, t.WorkspacePath, taskWorkDir(t), t.Forest)
}

func (o *Orchestrator) CreateTask(prompt, workspacePath string) (*Task, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, fmt.Errorf("任务内容不能为空")
	}
	id := newID("forest")
	title := titleFromPrompt(prompt)
	workspacePath = strings.TrimSpace(expandHome(workspacePath))
	if workspacePath == "" {
		workspacePath = defaultOutputPathForPrompt(prompt, id, title)
	}
	absWorkspace, err := filepath.Abs(workspacePath)
	if err != nil {
		return nil, err
	}
	if !o.isAllowedWorkspacePath(absWorkspace) {
		return nil, fmt.Errorf("保存位置不在允许的工作区范围内；如需使用外部目录，请设置 AUTO_GARDENER_ALLOWED_WORKSPACE_ROOTS")
	}
	if err := os.MkdirAll(absWorkspace, 0755); err != nil {
		return nil, fmt.Errorf("创建交付目录失败：%w", err)
	}
	scratchPath := filepath.Join(os.TempDir(), "GardenerScratch", id+"_"+safeName(title))
	absScratch, err := filepath.Abs(scratchPath)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absScratch, 0755); err != nil {
		return nil, fmt.Errorf("创建临时工作目录失败：%w", err)
	}
	settings := o.store.GetSettings()
	modelMode := normalizeModelMode(settings.ModelMode)
	cliEngine := compatibleCLIEngine(settings.CLIEngine, modelMode)
	forestDir := filepath.Join(o.dataDir, "forests", id)
	task := &Task{
		ID:                 id,
		Title:              title,
		Prompt:             prompt,
		WorkspacePath:      absWorkspace,
		ScratchPath:        absScratch,
		CLIEngine:          cliEngine,
		ModelMode:          modelMode,
		Status:             StatusRunning,
		GardenerStatus:     StatusRunning,
		MaxTreesPerForest:  o.maxTrees,
		MaxConcurrentTrees: o.maxConcurrent,
		SchedulePath:       filepath.Join(forestDir, "gardener", "schedule.md"),
		LogPath:            filepath.Join(forestDir, "gardener", "log.md"),
		Messages: []Message{
			{ID: newID("msg"), Role: RoleUser, Content: prompt, CreatedAt: time.Now()},
		},
	}
	if err := o.store.AddTask(task); err != nil {
		return nil, err
	}
	_ = o.store.WriteSchedule(id, initialSchedule(task))
	o.store.AppendGardenerLog(id, "任务创建，交付目录="+absWorkspace+"，临时工作目录="+absScratch)
	o.startForestRun(id, prompt)
	created, _ := o.store.GetTask(id)
	return created, nil
}

func (o *Orchestrator) isAllowedWorkspacePath(path string) bool {
	if os.Getenv("AUTO_GARDENER_ALLOW_ANY_WORKSPACE") == "1" {
		return true
	}
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return false
	}
	roots := o.allowedWorkspaceRoots()
	for _, root := range roots {
		rootAbs, err := filepath.Abs(filepath.Clean(expandHome(root)))
		if err != nil || strings.TrimSpace(rootAbs) == "" {
			continue
		}
		if abs == rootAbs || strings.HasPrefix(abs, rootAbs+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func (o *Orchestrator) allowedWorkspaceRoots() []string {
	var roots []string
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		roots = append(roots, home)
	}
	if strings.TrimSpace(o.dataDir) != "" {
		roots = append(roots, filepath.Join(o.dataDir, "workspaces"))
	}
	roots = append(roots, filepath.Join(os.TempDir(), "GardenerOutputs"))
	for _, root := range filepath.SplitList(os.Getenv("AUTO_GARDENER_ALLOWED_WORKSPACE_ROOTS")) {
		if strings.TrimSpace(root) != "" {
			roots = append(roots, root)
		}
	}
	return roots
}

func (o *Orchestrator) SendMessage(taskID, content string) (*Task, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("消息不能为空")
	}
	if snapshot, ok := o.store.GetTask(taskID); ok && snapshot.Status == StatusRunning && isProgressQuery(content) {
		now := time.Now()
		answer := buildProgressQueryAnswer(snapshot, now)
		t, err := o.store.UpdateTask(taskID, func(t *Task) {
			t.Messages = append(t.Messages,
				Message{ID: newID("msg"), Role: RoleUser, Content: content, CreatedAt: now},
				Message{ID: newID("msg"), Role: RoleGardener, Content: answer, CreatedAt: now},
			)
		})
		if err != nil {
			return nil, err
		}
		o.store.AppendGardenerLog(taskID, "用户查询进度：已只读回复，未中断正在运行的任务。")
		return t, nil
	}
	return o.sendMessageWithInstruction(taskID, content, content)
}

func (o *Orchestrator) ResumeTask(taskID string) (*Task, error) {
	t, ok := o.store.GetTask(taskID)
	if !ok {
		return nil, ErrNotFound
	}
	visible := "请继续这个任务。"
	return o.sendMessageWithInstruction(taskID, visible, buildContinueInstruction(t))
}

func (o *Orchestrator) sendMessageWithInstruction(taskID, visibleContent, instruction string) (*Task, error) {
	visibleContent = strings.TrimSpace(visibleContent)
	instruction = strings.TrimSpace(instruction)
	if visibleContent == "" {
		return nil, fmt.Errorf("消息不能为空")
	}
	if instruction == "" {
		instruction = visibleContent
	}
	t, err := o.store.UpdateTask(taskID, func(t *Task) {
		t.Messages = append(t.Messages, Message{ID: newID("msg"), Role: RoleUser, Content: visibleContent, CreatedAt: time.Now()})
		t.Status = StatusRunning
		t.GardenerStatus = StatusRunning
		t.StopRequested = false
	})
	if err != nil {
		return nil, err
	}
	o.store.AppendGardenerLog(taskID, "用户追加消息："+visibleContent)
	o.startForestRun(taskID, instruction)
	return t, nil
}

func isProgressQuery(content string) bool {
	text := strings.ToLower(strings.TrimSpace(content))
	if text == "" {
		return false
	}
	compact := strings.ReplaceAll(text, " ", "")
	// Avoid swallowing real follow-up instructions such as “修复状态管理 bug”.
	instructionWords := []string{"修复", "修改", "改成", "实现", "添加", "新增", "创建", "生成", "写", "删除", "重构", "fix", "change", "modify", "implement", "add", "create", "write", "delete", "refactor"}
	explicitProgressWords := []string{"进度", "做到哪", "到哪", "完成了吗", "好了没", "还在", "卡住", "有动静", "当前情况", "现在如何", "progress", "how is it", "how's it", "still running", "stuck", "any update", "update?"}
	for _, keyword := range explicitProgressWords {
		if strings.Contains(compact, strings.ReplaceAll(keyword, " ", "")) || strings.Contains(text, keyword) {
			return true
		}
	}
	statusOnlyWords := []string{"状态", "怎么样", "怎样了", "status"}
	for _, keyword := range statusOnlyWords {
		if !(strings.Contains(compact, strings.ReplaceAll(keyword, " ", "")) || strings.Contains(text, keyword)) {
			continue
		}
		for _, word := range instructionWords {
			if strings.Contains(compact, strings.ReplaceAll(word, " ", "")) || strings.Contains(text, word) {
				return false
			}
		}
		return len([]rune(text)) <= 80
	}
	return false
}

func buildProgressQueryAnswer(t *Task, now time.Time) string {
	if t == nil {
		return "我暂时没有找到这个任务的状态。"
	}
	var b strings.Builder
	if t.Status == StatusRunning {
		b.WriteString("当前任务仍在运行中。你这次只是查看进度，没有中断正在执行的工作。")
	} else {
		b.WriteString("当前任务已暂停或结束。如果你觉得还没完成，可以点击“继续任务”。")
	}
	if latest := latestHumanProgress(t); latest != "" {
		b.WriteString("\n\n最近进展：\n")
		b.WriteString(latest)
	}
	if !t.UpdatedAt.IsZero() {
		idle := now.Sub(t.UpdatedAt)
		if idle >= 5*time.Minute && t.Status == StatusRunning {
			b.WriteString(fmt.Sprintf("\n\n提示：界面已经约 %d 分钟没有收到新的输出。长任务可能仍在底层 CLI 中运行；如果后续自动暂停，我会在聊天区给出原因和继续入口。", int(idle.Minutes())))
		}
	}
	return b.String()
}

func latestHumanProgress(t *Task) string {
	var rows []string
	for i := len(t.GardenerProgress) - 1; i >= 0 && len(rows) < 3; i-- {
		line := strings.TrimSpace(t.GardenerProgress[i])
		if line != "" {
			rows = append(rows, "- "+line)
		}
	}
	for i := len(t.Trees) - 1; i >= 0 && len(rows) < 6; i-- {
		tr := t.Trees[i]
		if tr == nil || len(tr.Progress) == 0 {
			continue
		}
		line := strings.TrimSpace(tr.Progress[len(tr.Progress)-1])
		if line != "" {
			rows = append(rows, fmt.Sprintf("- %s：%s", tr.Name, line))
		}
	}
	return strings.Join(rows, "\n")
}

func (o *Orchestrator) StopTask(taskID string) (*Task, error) {
	o.store.AppendGardenerLog(taskID, "收到停止任务请求，正在中断 Gardener 和子任务的底层 CLI 进程。")
	o.cancelTaskProcesses(taskID)
	paths := map[string]string{}
	if snapshot, ok := o.store.GetTask(taskID); ok {
		now := time.Now()
		for _, tr := range snapshot.Trees {
			if tr.Status == StatusRunning && tr.FruitPath == "" {
				if path, err := o.writeFruit(snapshot, tr, "用户停止任务，子任务未完成正常执行。", fmt.Errorf("用户停止任务"), now, now); err == nil {
					paths[tr.ID] = path
				}
			}
		}
	}
	return o.store.UpdateTask(taskID, func(t *Task) {
		t.StopRequested = true
		t.Status = StatusFinished
		t.GardenerStatus = StatusFinished
		for _, tr := range t.Trees {
			if tr.Status == StatusRunning {
				tr.Status = StatusFinished
				now := time.Now()
				tr.CompletedAt = &now
				tr.Progress = append(tr.Progress, "任务被用户停止。")
				if paths[tr.ID] != "" {
					tr.FruitPath = paths[tr.ID]
				}
			}
		}
	})
}

func (o *Orchestrator) DeleteTask(taskID string) error {
	o.cancelTaskProcesses(taskID)
	return o.store.DeleteTask(taskID)
}

func (o *Orchestrator) RenameTask(taskID, title string) (*Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("任务名称不能为空")
	}
	runes := []rune(title)
	if len(runes) > 80 {
		title = string(runes[:80])
	}
	task, err := o.store.UpdateTask(taskID, func(t *Task) {
		t.Title = title
	})
	if err != nil {
		return nil, err
	}
	o.store.AppendGardenerLog(taskID, "任务重命名为："+title)
	return task, nil
}

func (o *Orchestrator) startForestRun(taskID, instruction string) {
	o.cancelTaskProcesses(taskID)
	ctx, cancel := context.WithCancel(context.Background())
	runID := newID("run")
	key := taskID + ":gardener:" + runID
	o.registerRun(taskID, runID, key, cancel)
	go func() {
		defer o.unregisterCancel(key)
		defer func() {
			if rec := recover(); rec != nil {
				o.store.AppendGardenerLog(taskID, fmt.Sprintf("Gardener 运行异常中断：%v", rec))
				o.appendSystemMessage(taskID, "任务运行过程中发生异常，已自动暂停，避免继续造成混乱。你可以点击“继续任务”，Gardener 会检查当前文件和已有报告后接着处理。")
				if o.isActiveRun(taskID, runID) {
					o.finishTask(taskID, "任务因运行异常已暂停。")
				}
			}
		}()
		o.runForest(ctx, taskID, instruction, runID)
	}()
}

func (o *Orchestrator) runForest(ctx context.Context, taskID, instruction, runID string) {
	if _, ok := o.store.GetTask(taskID); !ok {
		return
	}
	o.store.AppendGardenerLog(taskID, "Gardener 开始运行。")
	o.runGardenerGitInit(ctx, taskID)
	if ctx.Err() != nil || isStopRequested(o.store, taskID) {
		if o.isActiveRun(taskID, runID) {
			o.finishAfterStop(taskID)
		} else {
			o.store.AppendGardenerLog(taskID, "旧 Gardener run 被新的用户指令取代，已退出。")
		}
		return
	}

	plan := o.runGardenerPlan(ctx, taskID, instruction)
	for autoForest := 1; ; autoForest++ {
		if ctx.Err() != nil || isStopRequested(o.store, taskID) {
			if o.isActiveRun(taskID, runID) {
				o.finishAfterStop(taskID)
			} else {
				o.store.AppendGardenerLog(taskID, "旧 Gardener run 被新的用户指令取代，已退出。")
			}
			return
		}
		if strings.TrimSpace(plan.MessageToUser) != "" {
			_, _ = o.store.UpdateTask(taskID, func(t *Task) {
				t.Messages = append(t.Messages, Message{ID: newID("msg"), Role: RoleGardener, Content: userFacingMessage(plan.MessageToUser), CreatedAt: time.Now()})
			})
		}
		if plan.ForestFinished || len(plan.Trees) == 0 {
			if o.isActiveRun(taskID, runID) {
				o.finishTask(taskID, "这件事暂时处理完了。")
			}
			return
		}
		current, ok := o.store.GetTask(taskID)
		if !ok {
			return
		}
		if len(plan.Trees) > current.MaxTreesPerForest {
			plan.Trees = plan.Trees[:current.MaxTreesPerForest]
		}
		nextForest := current.Forest + 1
		_, _ = o.store.UpdateTask(taskID, func(t *Task) {
			t.Forest = nextForest
			t.Status = StatusRunning
			t.GardenerStatus = StatusRunning
		})
		o.appendSchedulePlan(taskID, nextForest, plan)
		o.runTreeForest(ctx, taskID, nextForest, plan.Trees)
		if ctx.Err() != nil || isStopRequested(o.store, taskID) {
			if o.isActiveRun(taskID, runID) {
				o.finishAfterStop(taskID)
			} else {
				o.store.AppendGardenerLog(taskID, "旧 Gardener run 被新的用户指令取代，已退出。")
			}
			return
		}
		o.runValidationTree(ctx, taskID, nextForest)
		if ctx.Err() != nil || isStopRequested(o.store, taskID) {
			if o.isActiveRun(taskID, runID) {
				o.finishAfterStop(taskID)
			} else {
				o.store.AppendGardenerLog(taskID, "旧 Gardener run 被新的用户指令取代，已退出。")
			}
			return
		}
		decision := o.runGardenerDecision(ctx, taskID, nextForest)
		if ctx.Err() != nil || isStopRequested(o.store, taskID) {
			if o.isActiveRun(taskID, runID) {
				o.finishAfterStop(taskID)
			} else {
				o.store.AppendGardenerLog(taskID, "旧 Gardener run 被新的用户指令取代，已退出。")
			}
			return
		}
		if decision.ForestFinished || len(decision.Trees) == 0 {
			if strings.TrimSpace(decision.MessageToUser) != "" {
				_, _ = o.store.UpdateTask(taskID, func(t *Task) {
					t.Messages = append(t.Messages, Message{ID: newID("msg"), Role: RoleGardener, Content: codex.Truncate(userFacingMessage(decision.MessageToUser), 900), CreatedAt: time.Now()})
				})
			}
			if o.isActiveRun(taskID, runID) {
				o.finishTask(taskID, fmt.Sprintf("第 %d 个阶段已完成。", nextForest))
			}
			return
		}
		plan = decision
		o.store.AppendGardenerLog(taskID, fmt.Sprintf("Gardener 决定继续派出下一批子任务修复或补充，autoStage=%d。", autoForest+1))
	}
}

func (o *Orchestrator) runGardenerGitInit(ctx context.Context, taskID string) {
	t, ok := o.store.GetTask(taskID)
	if !ok {
		return
	}
	if filepath.Clean(taskWorkDir(t)) != filepath.Clean(t.WorkspacePath) {
		o.store.AppendGardenerLog(taskID, "已启用临时工作目录隔离，跳过交付目录 Git 初始化，避免向用户目录写入过程文件。")
		return
	}
	outFile := filepath.Join(o.dataDir, "forests", taskID, "gardener", "gardener_git_init.md")
	usage := o.newUsageRecorder(taskID, newID("agent"), "gardener", "", "Gardener Git Init")
	prompt := fmt.Sprintf(`你是 Gardener 的 Gardener。请在当前 workspace 中完成 Git 统一初始化。

规则：
- 你的工作目录是 workspacePath。
- 如果 workspacePath 不是 Git 仓库，请执行 git init。
- 如果目录中已有文件，请尝试执行初始提交：git add . && git commit -m "Initial commit by Gardener"。
- 如果目录为空，只需要 git init。
- 如果 commit 因用户名/邮箱或其他原因失败，不要阻塞后续任务；请在最终报告中说明失败原因。
- 允许你自主执行所有必要命令。

workspacePath: %s
`, t.WorkspacePath)
	result := o.runner.Run(ctx, codex.RunRequest{
		Role:       "gardener",
		CLI:        string(normalizeCLIEngine(t.CLIEngine)),
		Prompt:     prompt,
		WorkDir:    taskWorkDir(t),
		OutputFile: outFile,
		Model:      o.modelConfigForTask(t),
		OnLine: func(line string) {
			usage.Record(line)
			if cleaned, ok := o.codexLogLine(line); ok {
				o.store.AppendGardenerLog(taskID, "初始化："+cleaned)
			}
		},
	})
	if result.Err != nil {
		o.store.AppendGardenerLog(taskID, "Gardener Git 初始化阶段发生错误，但按需求继续后续规划："+result.Err.Error())
	}
	if strings.TrimSpace(result.Output) != "" {
		o.store.AppendGardenerLog(taskID, "Gardener Git 初始化输出摘要："+codex.Truncate(result.Output, 1200))
	}
}

func (o *Orchestrator) runGardenerPlan(ctx context.Context, taskID, instruction string) GardenerPlan {
	t, ok := o.store.GetTask(taskID)
	if !ok {
		return GardenerPlan{ForestFinished: true}
	}
	outFile := filepath.Join(o.dataDir, "forests", taskID, "gardener", fmt.Sprintf("gardener_plan_forest_%d.md", t.Forest+1))
	prompt := buildGardenerPlanPrompt(t, instruction)
	usage := o.newUsageRecorder(taskID, newID("agent"), "gardener", "", "Gardener Plan")
	result := o.runner.Run(ctx, codex.RunRequest{
		Role:       "gardener",
		CLI:        string(normalizeCLIEngine(t.CLIEngine)),
		Prompt:     prompt,
		WorkDir:    taskWorkDir(t),
		OutputFile: outFile,
		Model:      o.modelConfigForTask(t),
		OnLine: func(line string) {
			usage.Record(line)
			if cleaned, ok := o.codexLogLine(line); ok {
				o.store.AppendGardenerLog(taskID, "规划："+cleaned)
			}
		},
	})
	if result.Err != nil {
		msg := "Gardener 规划失败，未创建任何子任务：" + result.Err.Error()
		if ctx.Err() != nil {
			o.store.AppendGardenerLog(taskID, msg+"；该 run 已被新的用户指令或继续任务请求取消，不向用户显示模型失败。")
			return GardenerPlan{ForestFinished: true}
		}
		o.store.AppendGardenerLog(taskID, msg)
		o.appendSystemMessage(taskID, "本次请求没有完成：底层 CLI 或模型连接失败。请检查设置中的 CLI / 模型配置后，点击“继续任务”重试。")
		return GardenerPlan{ForestFinished: true}
	}
	o.store.AppendGardenerLog(taskID, "Gardener 规划原始输出已保存。")
	plan, err := parsePlan(result.Output)
	if err != nil {
		if ctx.Err() != nil {
			o.store.AppendGardenerLog(taskID, "Gardener 规划输出解析前 run 已取消，不向用户显示格式异常。")
			return GardenerPlan{ForestFinished: true}
		}
		msg := "Gardener 输出不是有效调度 JSON，未创建任何子任务：" + err.Error()
		o.store.AppendGardenerLog(taskID, msg)
		o.store.AppendGardenerLog(taskID, "无效 Gardener 输出摘要："+codex.Truncate(result.Output, 1200))
		o.appendSystemMessage(taskID, "规划结果格式异常，任务已暂停。通常点击“继续任务”即可让 Gardener 重新检查当前文件并继续。")
		return GardenerPlan{ForestFinished: true}
	}
	return normalizePlan(plan, t, instruction)
}

func (o *Orchestrator) runGardenerDecision(ctx context.Context, taskID string, forest int) GardenerPlan {
	t, ok := o.store.GetTask(taskID)
	if !ok {
		return GardenerPlan{ForestFinished: true}
	}
	outFile := filepath.Join(o.dataDir, "forests", taskID, "gardener", fmt.Sprintf("gardener_decision_forest_%d.md", forest))
	usage := o.newUsageRecorder(taskID, newID("agent"), "gardener", "", fmt.Sprintf("Gardener Decision O%d", forest))
	result := o.runner.Run(ctx, codex.RunRequest{
		Role:       "gardener",
		CLI:        string(normalizeCLIEngine(t.CLIEngine)),
		Prompt:     buildGardenerDecisionPrompt(t, forest),
		WorkDir:    taskWorkDir(t),
		OutputFile: outFile,
		Model:      o.modelConfigForTask(t),
		OnLine: func(line string) {
			usage.Record(line)
			if cleaned, ok := o.codexLogLine(line); ok {
				o.store.AppendGardenerLog(taskID, "判断："+cleaned)
			}
		},
	})
	if result.Err != nil {
		if ctx.Err() != nil {
			o.store.AppendGardenerLog(taskID, "Gardener 决策 run 已被新的用户指令或继续任务请求取消："+result.Err.Error())
			return GardenerPlan{ForestFinished: true}
		}
		o.store.AppendGardenerLog(taskID, "Gardener 决策失败："+result.Err.Error())
		o.appendSystemMessage(taskID, "后续判断没有完成：底层 CLI 或模型连接失败。请检查设置中的 CLI / 模型配置后，点击“继续任务”重试。")
		return GardenerPlan{ForestFinished: true}
	}
	plan, err := parsePlan(result.Output)
	if err != nil {
		if ctx.Err() != nil {
			o.store.AppendGardenerLog(taskID, "Gardener 决策输出解析前 run 已取消，不向用户显示格式异常。")
			return GardenerPlan{ForestFinished: true}
		}
		o.store.AppendGardenerLog(taskID, "解析 Gardener 决策 JSON 失败，默认任务 Finished："+err.Error())
		o.appendSystemMessage(taskID, "后续判断结果格式异常，任务已暂停。你可以点击“继续任务”，Gardener 会重新检查当前进度并继续。")
		return GardenerPlan{ForestFinished: true}
	}
	return normalizePlan(plan, t, "验证后续决策")
}

func (o *Orchestrator) runTreeForest(ctx context.Context, taskID string, forest int, plans []TreePlan) {
	sem := make(chan struct{}, o.maxConcurrent)
	var wg sync.WaitGroup
	for _, p := range plans {
		if ctx.Err() != nil || isStopRequested(o.store, taskID) {
			break
		}
		tr := &Tree{
			ID:        newID("tree"),
			TaskID:    taskID,
			Forest:    forest,
			Name:      p.Name,
			Objective: p.Objective,
			Prompt:    p.Prompt,
			Scope:     p.Scope,
			Status:    StatusRunning,
			Progress:  []string{"已创建，等待并行执行槽。"},
		}
		tr.GoalPath = o.treeGoalPath(taskID, tr.ID)
		_, _ = o.store.AddTree(taskID, tr)
		wg.Add(1)
		go func(treeID string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				o.finishTreeBeforeRun(taskID, treeID, "子任务在开始前被停止。")
				return
			}
			o.runTree(ctx, taskID, treeID)
		}(tr.ID)
	}
	wg.Wait()
}

func (o *Orchestrator) finishTreeBeforeRun(taskID, treeID, reason string) {
	task, ok := o.store.GetTask(taskID)
	if !ok {
		return
	}
	tr, err := o.store.FindTree(taskID, treeID)
	if err != nil {
		return
	}
	now := time.Now()
	path, _ := o.writeFruit(task, tr, reason, fmt.Errorf(reason), now, now)
	_, _ = o.store.UpdateTree(taskID, treeID, func(tr *Tree) {
		tr.Status = StatusFinished
		tr.CompletedAt = &now
		tr.FruitPath = path
		tr.Progress = append(tr.Progress, reason)
	})
}

func (o *Orchestrator) runValidationTree(ctx context.Context, taskID string, forest int) {
	t, ok := o.store.GetTask(taskID)
	if !ok {
		return
	}
	tr := &Tree{
		ID:           newID("tree"),
		TaskID:       taskID,
		Forest:       forest,
		Name:         fmt.Sprintf("验证子任务 %d", forest),
		Objective:    "检查本轮成果是否完整、是否有冲突、是否可以交付。",
		Prompt:       buildValidationPrompt(t, forest),
		Scope:        []string{"全项目验证", "测试/构建", "冲突与风险检查"},
		IsValidation: true,
		Status:       StatusRunning,
		Progress:     []string{"验证子任务已准备。"},
	}
	tr.GoalPath = o.treeGoalPath(taskID, tr.ID)
	_, _ = o.store.AddTree(taskID, tr)
	o.runTree(ctx, taskID, tr.ID)
}

func (o *Orchestrator) runTree(ctx context.Context, taskID, treeID string) {
	tr, err := o.store.FindTree(taskID, treeID)
	if err != nil {
		return
	}
	task, ok := o.store.GetTask(taskID)
	if !ok {
		return
	}
	treeCtx, cancel := context.WithCancel(ctx)
	key := taskID + ":" + treeID
	o.registerCancel(key, cancel)
	defer o.unregisterCancel(key)
	defer cancel()

	start := time.Now()
	_, _ = o.store.UpdateTree(taskID, treeID, func(tr *Tree) {
		tr.Status = StatusRunning
		tr.StartedAt = &start
		if strings.TrimSpace(tr.GoalPath) == "" {
			tr.GoalPath = o.treeGoalPath(taskID, treeID)
		}
		tr.Progress = append(tr.Progress, "开始处理。")
	})
	if refreshed, err := o.store.FindTree(taskID, treeID); err == nil {
		tr = refreshed
	}
	_ = o.writeTreeGoal(task, tr, "Running", start, nil, "子任务 goal 已创建，底层 CLI 将以 goal 模式执行。", "")
	o.store.AppendTreeProgress(taskID, treeID, "开始处理。")
	outFile := filepath.Join(o.dataDir, "forests", taskID, "trees", treeID, "agent_last_message.md")
	usage := o.newUsageRecorder(taskID, newID("agent"), "tree", treeID, tr.Name)
	result := o.runner.Run(treeCtx, codex.RunRequest{
		Role:       "tree",
		CLI:        string(normalizeCLIEngine(task.CLIEngine)),
		Prompt:     buildTreePrompt(task, tr),
		WorkDir:    taskWorkDir(task),
		OutputFile: outFile,
		Model:      o.modelConfigForTask(task),
		Goal:       o.treeGoalSpec(task, tr),
		OnLine: func(line string) {
			usage.Record(line)
			if cleaned, ok := o.codexLogLine(line); ok {
				o.store.AppendTreeProgress(taskID, treeID, cleaned)
			}
		},
	})
	end := time.Now()
	if treeCtx.Err() != nil && result.Err == nil {
		result.Err = treeCtx.Err()
	}
	fruitPath, fruitErr := o.writeFruit(task, tr, result.Output, result.Err, start, end)
	if fruitErr != nil {
		o.store.AppendTreeProgress(taskID, treeID, "保存成果失败："+fruitErr.Error())
	}
	goalStatus, goalNote := inferGoalStatus(result.Output, result.Err)
	_ = o.writeTreeGoal(task, tr, goalStatus, start, &end, goalNote, result.Output)
	_, _ = o.store.UpdateTree(taskID, treeID, func(tr *Tree) {
		tr.Status = StatusFinished
		tr.CompletedAt = &end
		tr.FruitPath = fruitPath
		if result.Err != nil {
			tr.Progress = append(tr.Progress, "处理结束，但有异常："+result.Err.Error())
		} else {
			tr.Progress = append(tr.Progress, "处理完成，成果已生成。")
		}
	})
	o.store.AppendTreeProgress(taskID, treeID, "已完成。")
}

func (o *Orchestrator) modelConfigForTask(t *Task) codex.ModelConfig {
	settings := o.store.GetSettings()
	mode := ModelModeDefault
	if t != nil {
		mode = normalizeModelMode(t.ModelMode)
	}
	switch mode {
	case ModelModeMiniMax:
		return codex.ModelConfig{
			ProviderID:   "gardener-minimax",
			ProviderName: "Gardener MiniMax Compatibility",
			Model:        firstNonEmpty(os.Getenv("AUTO_GARDENER_MINIMAX_MODEL"), "MiniMax-M2.7-highspeed"),
			BaseURL:      firstNonEmpty(os.Getenv("AUTO_GARDENER_MINIMAX_BASE_URL"), o.compatProviderBaseURL("minimax")),
			EnvKey:       "GARDENER_MINIMAX_API_KEY",
			Token:        settings.MiniMaxToken,
			WireAPI:      "responses",
		}
	case ModelModeKimi:
		return codex.ModelConfig{
			ProviderID:   "gardener-kimi",
			ProviderName: "Gardener Kimi Compatibility",
			Model:        firstNonEmpty(os.Getenv("AUTO_GARDENER_KIMI_MODEL"), "kimi-coding"),
			BaseURL:      firstNonEmpty(os.Getenv("AUTO_GARDENER_KIMI_BASE_URL"), o.compatProviderBaseURL("kimi")),
			EnvKey:       "GARDENER_KIMI_API_KEY",
			Token:        settings.KimiToken,
			WireAPI:      "responses",
		}
	default:
		return codex.ModelConfig{}
	}
}

func (o *Orchestrator) compatProviderBaseURL(provider string) string {
	if o.compatBaseURL == "" {
		return ""
	}
	return o.compatBaseURL + "/" + provider + "/v1"
}

func (o *Orchestrator) appendSystemMessage(taskID, content string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}
	_, _ = o.store.UpdateTask(taskID, func(t *Task) {
		t.Messages = append(t.Messages, Message{ID: newID("msg"), Role: RoleSystem, Content: content, CreatedAt: time.Now()})
	})
}

func (o *Orchestrator) finishAfterStop(taskID string) {
	o.finishTask(taskID, "已停止。")
}

func (o *Orchestrator) finishTask(taskID, message string) {
	o.store.AppendGardenerLog(taskID, message)
	_, _ = o.store.UpdateTask(taskID, func(t *Task) {
		t.Status = StatusFinished
		t.GardenerStatus = StatusFinished
	})
}

func (o *Orchestrator) appendSchedulePlan(taskID string, forest int, plan GardenerPlan) {
	t, ok := o.store.GetTask(taskID)
	if !ok {
		return
	}
	var b strings.Builder
	if existing, err := os.ReadFile(t.SchedulePath); err == nil {
		b.Write(existing)
		if !strings.HasSuffix(b.String(), "\n") {
			b.WriteByte('\n')
		}
	}
	b.WriteString(fmt.Sprintf("\n## 阶段 %d - %s\n\n", forest, time.Now().Format(time.RFC3339)))
	b.WriteString("### Gardener 给用户的说明\n\n")
	b.WriteString(plan.MessageToUser + "\n\n")
	b.WriteString("### 子任务调度计划\n\n")
	for i, tr := range plan.Trees {
		b.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, tr.Name))
		b.WriteString("   - 目标：" + tr.Objective + "\n")
		b.WriteString("   - 范围：" + strings.Join(tr.Scope, ", ") + "\n")
	}
	_ = o.store.WriteSchedule(taskID, b.String())
}

func (o *Orchestrator) treeGoalPath(taskID, treeID string) string {
	return filepath.Join(o.dataDir, "forests", taskID, "trees", treeID, "goal.md")
}

func (o *Orchestrator) treeGoalSpec(task *Task, tr *Tree) codex.GoalSpec {
	if task == nil || tr == nil {
		return codex.GoalSpec{}
	}
	goalPath := strings.TrimSpace(tr.GoalPath)
	if goalPath == "" {
		goalPath = o.treeGoalPath(task.ID, tr.ID)
	}
	criteria := []string{
		"只完成本子任务目标，不越界处理其他子任务职责。",
		"在 workspace 中完成必要的文件修改或检查，并保留可追溯证据。",
		"最终报告明确说明 goal 状态、已改文件、验证方式、风险和下一步建议。",
	}
	if tr.IsValidation {
		criteria = []string{
			"检查本阶段普通子任务成果是否完整、冲突是否解决。",
			"运行可行的测试、构建或静态检查，并记录结果。",
			"明确给出是否可以交付，以及仍需修复的具体问题。",
		}
	}
	for _, scope := range tr.Scope {
		scope = strings.TrimSpace(scope)
		if scope != "" {
			criteria = append(criteria, "重点范围："+scope)
		}
	}
	return codex.GoalSpec{
		ID:              tr.ID,
		Title:           humanGoalTitle(tr),
		Objective:       strings.TrimSpace(tr.Objective),
		SuccessCriteria: criteria,
		Path:            goalPath,
	}
}

func inferGoalStatus(output string, runErr error) (string, string) {
	if runErr != nil {
		return "blocked", "子任务运行结束但存在异常；需要 Gardener 或后续子任务检查。"
	}
	lower := strings.ToLower(strings.TrimSpace(output))
	blockedHints := []string{
		"goal status: blocked", "status: blocked", "blocked", "cannot complete", "can't complete", "unable to complete",
		"too large", "too big", "out of scope", "need more context", "缺少", "无法完成", "不能完成", "阻塞", "被阻塞", "任务量太大", "不建议继续", "需要用户", "需要更多",
	}
	for _, hint := range blockedHints {
		if strings.Contains(lower, hint) {
			return "blocked", "底层 CLI 表示 goal 未能完成或已被阻塞；请查看报告中的原因和下一步建议。"
		}
	}
	partialHints := []string{"goal status: partial", "status: partial", "partial", "partially", "部分完成", "未全部完成", "尚未完成"}
	for _, hint := range partialHints {
		if strings.Contains(lower, hint) {
			return "partial", "底层 CLI 表示 goal 仅部分完成；需要 Gardener 判断是否继续派发后续子任务。"
		}
	}
	return "complete", "子任务已结束，成果报告已生成。"
}

func humanGoalTitle(tr *Tree) string {
	if tr == nil {
		return "完成子任务"
	}
	name := strings.TrimSpace(tr.Name)
	if name == "" {
		name = tr.ID
	}
	if tr.IsValidation {
		return "验证子任务：" + name
	}
	return "完成子任务：" + name
}

func (o *Orchestrator) writeTreeGoal(task *Task, tr *Tree, status string, start time.Time, end *time.Time, note, output string) error {
	if task == nil || tr == nil {
		return nil
	}
	goal := o.treeGoalSpec(task, tr)
	path := strings.TrimSpace(goal.Path)
	if path == "" {
		return nil
	}
	if err := ensurePrivateDir(filepath.Dir(path)); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# 子任务 Goal\n\n")
	b.WriteString("## Goal 元数据\n\n")
	b.WriteString("- Goal ID: " + goal.ID + "\n")
	b.WriteString("- Goal 标题: " + goal.Title + "\n")
	b.WriteString("- 所属任务 ID: " + task.ID + "\n")
	b.WriteString("- 所属任务: " + task.Title + "\n")
	b.WriteString("- 阶段: " + strconv.Itoa(tr.Forest) + "\n")
	b.WriteString("- 状态: " + status + "\n")
	b.WriteString("- 开始时间: " + start.Format(time.RFC3339) + "\n")
	if end != nil {
		b.WriteString("- 结束时间: " + end.Format(time.RFC3339) + "\n")
	}
	b.WriteString("- 是否验证子任务: " + strconv.FormatBool(tr.IsValidation) + "\n")
	b.WriteString("- 工作区: " + task.WorkspacePath + "\n")
	b.WriteString("\n## Goal 目标\n\n")
	b.WriteString(strings.TrimSpace(goal.Objective) + "\n")
	b.WriteString("\n## 验收标准\n\n")
	for _, item := range goal.SuccessCriteria {
		item = strings.TrimSpace(item)
		if item != "" {
			b.WriteString("- " + item + "\n")
		}
	}
	b.WriteString("\n## 当前说明\n\n")
	b.WriteString(strings.TrimSpace(note) + "\n")
	if strings.TrimSpace(output) != "" {
		b.WriteString("\n## CLI 输出摘要\n\n")
		b.WriteString(codex.Truncate(strings.TrimSpace(output), 2000) + "\n")
	}
	return writePrivateFile(path, []byte(b.String()))
}

func (o *Orchestrator) writeFruit(task *Task, tr *Tree, output string, runErr error, start, end time.Time) (string, error) {
	dir := filepath.Join(o.dataDir, "forests", task.ID, "trees", tr.ID)
	if err := ensurePrivateDir(dir); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "fruit.md")
	errText := ""
	if runErr != nil {
		errText = runErr.Error()
	}
	body := fmt.Sprintf(`# 子任务工作报告

## 1. 子任务基本信息

- 子任务 ID: %s
- 所属任务 ID: %s
- 所属任务: %s
- 子任务名称: %s
- 状态: Finished
- 开始时间: %s
- 结束时间: %s
- 交付目录: %s
- 临时工作目录: %s
- 工作范围: %s
- 是否验证子任务: %v

## 2. 子任务目标

%s

## 3. 执行过程

详见同目录 progress.log。底层 CLI 输出摘要如下。

## 4. 完成结果

%s

## 5. 产出文件或关键修改

由当前任务固定选择的底层 CLI 在临时工作目录执行；用户可见文件应只包含 outputPath 中的最终交付物或必要最终修改。

## 6. 遇到的问题

%s

## 7. 对 Gardener 的汇报

%s

## 8. 后续建议

由 Gardener 读取本报告和验证报告后决定是否派出新的子任务修复或继续。
`, tr.ID, task.ID, task.Title, tr.Name, start.Format(time.RFC3339), end.Format(time.RFC3339), task.WorkspacePath, taskWorkDir(task), strings.Join(tr.Scope, ", "), tr.IsValidation, tr.Objective, strings.TrimSpace(output), errText, codex.Truncate(output, 1200))
	return path, writePrivateFile(path, []byte(body))
}

func taskWorkDir(t *Task) string {
	if t == nil {
		return "."
	}
	if strings.TrimSpace(t.ScratchPath) != "" {
		return t.ScratchPath
	}
	return t.WorkspacePath
}

func defaultOutputPathForPrompt(prompt, id, title string) string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(os.TempDir(), "GardenerOutputs", id+"_"+safeName(title))
	}
	desktop := filepath.Join(home, "Desktop")
	text := strings.ToLower(prompt)
	if strings.Contains(prompt, "桌面") || strings.Contains(text, "desktop") {
		return desktop
	}
	return filepath.Join(desktop, "Gardener成果", id+"_"+safeName(title))
}

func buildGardenerPlanPrompt(t *Task, instruction string) string {
	return fmt.Sprintf(`你是 Gardener，一个本地 AI 任务编排助手。你负责理解用户目标、规划任务阶段，并把具体工作交给多个子任务执行；你自己不直接实现具体工作。

用户侧术语规则：
- 对用户说话时使用专业工作语言：任务、阶段、子任务、验证、报告、文件、工作区、进展。
- 产品名和主控助手可以继续叫 Gardener。
- 不要使用 Forest、Tree、Fruit、Orchard 等花园隐喻面向用户表达。
- 禁止使用“执行小队”“检查小队”“小队”“团队”等称呼。
- 内部 JSON 字段名仍然使用 trees / forest_finished，这是系统兼容字段；字段含义分别是“子任务列表”和“任务是否完成”。

关键规则：
- 任务/Gardener/子任务状态只有 Running 和 Finished。
- 底层 CLI 的当前工作目录是 scratchPath；这是临时工作目录，可用于搜索、下载缓存、草稿、脚本和中间文件。
- outputPath 是用户可见的交付目录；除非用户明确要求保存某个文件，否则不要要求子任务在 outputPath 或 Gardener 数据目录中创建过程文件。
- 只有最终交付物、用户明确要求保存的文件、或对用户项目的必要最终修改，才可以放入 outputPath。
- 多个子任务可能并行运行；你必须明确每个子任务的交付范围，尽量避免冲突。
- 如果冲突发生，后续由你派修复子任务处理。
- 每个阶段最多 %d 个普通子任务。
- 每个阶段的普通子任务完成后，系统会自动派验证子任务。
- 子任务可以实际修改文件、运行命令。
- 底层 CLI 被允许自主执行所有命令。
- message_to_user 是你作为 Gardener 对用户说的话，必须自然、克制、面向用户，不要出现工程状态口吻。

任务 ID: %s
任务标题: %s
原始任务: %s
本次用户指令: %s
outputPath: %s
scratchPath: %s
schedule.md: %s
log.md: %s

现有子任务摘要：
%s

请仅输出一个 JSON 对象，不要 Markdown，不要代码块。JSON 格式：
{
  "message_to_user": "Gardener 给用户的简短说明",
  "forest_finished": false,
  "trees": [
    {
      "name": "研究 / 实现 / 写作 / 修复 / 整合 等专业子任务名称",
      "objective": "该子任务的目标",
      "prompt": "给子任务的完整执行指令，必须包含只能负责本子任务和允许实际修改 workspace 文件；不得使用小队称呼；不要使用花园隐喻",
      "scope": ["建议负责的文件或目录范围"]
    }
  ]
}
`, t.MaxTreesPerForest, t.ID, t.Title, t.Prompt, instruction, t.WorkspacePath, taskWorkDir(t), t.SchedulePath, t.LogPath, treeSummary(t))
}

func buildTreePrompt(task *Task, tr *Tree) string {
	return fmt.Sprintf(`你是 Gardener 派出的一个子任务执行 agent。你背后是当前任务固定选择的底层 CLI agent，你可以实际修改文件，也可以运行需要的命令。

严格规则：
- 你必须以 goal 模式执行：开局创建/激活一个只属于本子任务的 goal，围绕该 goal 工作，并在最终报告中明确 goal status=complete/partial/blocked。
- 如果当前 Codex/Claude CLI 提供原生 goal 工具或 goal 模式，请优先使用原生能力记录目标、进展和完成状态；如果没有原生工具，则严格按本提示中的 Goal 协议执行。
- 如果任务过大、上下文不足、依赖缺失或无法安全完成，不要沉默停止；请标记 partial/blocked，并给出下一步建议。
- 你是子任务，不是小队、团队或执行小队；报告中也不要使用“小队”称呼。
- 你只负责自己的子任务，不要越界处理其他子任务的职责。
- 你的当前工作目录是 scratchPath；这里仅用于临时搜索、下载缓存、草稿、脚本和中间文件。
- outputPath 是用户可见的交付目录。除非用户明确要求保存某个文件，否则不要在 outputPath 或 Gardener 数据目录中创建过程文件。
- 只有最终交付物、用户明确要求保存的文件、或对用户项目的必要最终修改，才可以放入 outputPath。
- 如果用户明确说“下载到桌面 / 保存到桌面 / save to Desktop”，最终文件必须保存到 outputPath 指向的桌面目录，临时文件仍留在 scratchPath。
- 多个子任务可能并行运行；请尽量只改你的工作范围。
- 如果发现冲突或其他子任务造成的问题，请记录到报告中，不要大范围重写无关部分。
- 完成后请在最终回答中给出 Markdown 报告，系统会写入内部报告文件。
- 报告标题和正文应使用专业工作语言：任务、阶段、子任务、验证、报告、文件、工作区、进展；不要使用 Forest/Tree/Fruit 等隐喻。

任务 ID: %s
任务: %s
outputPath: %s
scratchPath: %s
子任务 ID: %s
子任务名称: %s
Goal 记录文件: %s
是否验证子任务: %v
工作范围: %s
子任务目标: %s

最终报告必须包含：
- Goal status: complete / partial / blocked
- 完成证据
- 修改/检查过的文件
- 运行过的验证命令及结果
- 风险和下一步建议

Gardener 给你的完整指令：
%s
`, task.ID, task.Prompt, task.WorkspacePath, taskWorkDir(task), tr.ID, tr.Name, tr.GoalPath, tr.IsValidation, strings.Join(tr.Scope, ", "), tr.Objective, tr.Prompt)
}

func buildValidationPrompt(t *Task, forest int) string {
	return fmt.Sprintf(`你是 Gardener 派出的验证子任务。请验证第 %d 阶段中各子任务的并行修改。

命名规则：
- 你必须以 goal 模式执行验证：开局创建/激活验证 goal，结束时明确 goal status=complete/partial/blocked。
- 如果当前 Codex/Claude CLI 提供原生 goal 工具或 goal 模式，请优先使用原生能力；如果没有，则严格按最终报告的 Goal status 协议执行。
- 你是验证子任务，不是检查小队或测试小队。
- 全文使用专业工作语言：任务、阶段、子任务、验证、报告、文件、工作区、进展。
- 禁止使用 Forest、Tree、Fruit、Orchard 等花园隐喻。
- 禁止使用“执行小队”“检查小队”“小队”“团队”等称呼。

你可以实际运行测试、构建、静态检查，也可以进行小范围必要修复；但你的主要职责是验证、发现冲突和总结风险。临时过程文件只能放在 scratchPath，用户可见的 outputPath 只放最终交付物或必要最终修改。

请检查：
1. 是否有明显冲突或互相覆盖。
2. 项目是否能构建/测试。
3. 哪些文件被关键修改。
4. 是否需要 Gardener 派新的修复子任务。

任务 ID: %s
任务: %s
outputPath: %s
scratchPath: %s

已有子任务摘要：
%s
`, forest, t.ID, t.Prompt, t.WorkspacePath, taskWorkDir(t), treeSummary(t))
}

func buildGardenerDecisionPrompt(t *Task, forest int) string {
	return fmt.Sprintf(`你是 Gardener。第 %d 阶段的普通子任务和验证子任务已经结束。

请读取当前交付目录、schedule.md、log.md、子任务报告和 progress.log，判断是否需要继续派新子任务修复冲突/补充实现，或是否任务可以 Finished。

用户侧术语规则：
- 对用户说话时使用专业工作语言：任务、阶段、子任务、验证、报告、文件、工作区、进展。
- 不要使用 Forest、Tree、Fruit、Orchard 等花园隐喻。
- 禁止使用“执行小队”“检查小队”“小队”“团队”等称呼。
- 如果需要修复，请派修复子任务；如果需要补充调研/写作/实现，可派研究/写作/实现子任务。
- message_to_user 是 Gardener 对用户的自然汇报，不要出现工程状态口吻。

重要规则：
- 如果验证子任务发现冲突、测试失败、明显缺口，你应该派新的修复子任务，并明确范围。
- 如果任务已经足够完成，forest_finished=true 且 trees=[]。
- 状态字段只有 Running 和 Finished；失败/停止/风险只写文本。
- 请仅输出 JSON 对象，不要 Markdown，不要代码块。

JSON 格式：
{
  "message_to_user": "Gardener 给用户的简短汇报",
  "forest_finished": true,
  "trees": [
    {"name":"修复 / 研究 / 实现 / 写作 等专业子任务名称","objective":"目标","prompt":"完整执行指令，且不得使用小队称呼或花园隐喻","scope":["负责范围"]}
  ]
}

任务 ID: %s
任务: %s
outputPath: %s
scratchPath: %s
schedule.md: %s
log.md: %s
子任务摘要：
%s
`, forest, t.ID, t.Prompt, t.WorkspacePath, taskWorkDir(t), t.SchedulePath, t.LogPath, treeSummary(t))
}

func initialSchedule(t *Task) string {
	return fmt.Sprintf(`# Gardener Schedule

- Task ID: %s
- Title: %s
- Output Path: %s
- Scratch Path: %s
- Status: Running
- Created At: %s

## 调度原则

- Gardener 和子任务均真实调用当前任务固定选择的底层 CLI。
- 底层 CLI 在临时工作目录 scratchPath 中运行；用户可见的 outputPath 只放最终交付物或必要最终修改。
- 多个子任务可能并行运行，需尽量避免交付冲突。
- Gardener 必须明确各子任务工作范围；冲突由 Gardener 后续派新子任务修复。
- 每个阶段的普通子任务完成后派验证子任务。

`, t.ID, t.Title, t.WorkspacePath, taskWorkDir(t), time.Now().Format(time.RFC3339))
}

func parsePlan(s string) (GardenerPlan, error) {
	var p GardenerPlan
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end < start {
		return p, fmt.Errorf("未找到 JSON 对象")
	}
	if err := json.Unmarshal([]byte(s[start:end+1]), &p); err != nil {
		return p, err
	}
	return p, nil
}

func normalizePlan(p GardenerPlan, t *Task, instruction string) GardenerPlan {
	for i := range p.Trees {
		if strings.TrimSpace(p.Trees[i].Name) == "" {
			p.Trees[i].Name = fmt.Sprintf("Tree %d", i+1)
		}
		if strings.TrimSpace(p.Trees[i].Objective) == "" {
			p.Trees[i].Objective = p.Trees[i].Name
		}
		if len(p.Trees[i].Scope) == 0 {
			p.Trees[i].Scope = []string{"相关文件"}
		}
		if strings.TrimSpace(p.Trees[i].Prompt) == "" {
			p.Trees[i].Prompt = fmt.Sprintf("你是 Gardener 派出的子任务执行 agent。请完成子任务：%s。本次用户指令：%s。报告中使用任务、阶段、子任务、验证、报告、文件、工作区、进展等专业工作语言，不要使用 Forest/Tree/Fruit 等隐喻，也不要使用小队称呼。", p.Trees[i].Objective, instruction)
		}
	}
	return p
}

func treeSummary(t *Task) string {
	if len(t.Trees) == 0 {
		return "暂无子任务。"
	}
	var b strings.Builder
	for _, tr := range t.Trees {
		b.WriteString(fmt.Sprintf("- %s / %s / 阶段 %d / Status %s / report: %s / scope: %s\n", tr.ID, tr.Name, tr.Forest, tr.Status, tr.FruitPath, strings.Join(tr.Scope, ", ")))
	}
	return b.String()
}

func isStopRequested(store *Store, taskID string) bool {
	t, ok := store.GetTask(taskID)
	return ok && t.StopRequested
}

func (o *Orchestrator) registerRun(taskID, runID, key string, cancel context.CancelFunc) {
	o.mu.Lock()
	o.activeRuns[taskID] = runID
	o.cancels[key] = cancel
	o.mu.Unlock()
}

func (o *Orchestrator) isActiveRun(taskID, runID string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.activeRuns[taskID] == runID
}

func (o *Orchestrator) cancelTaskProcesses(taskID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	for key, cancel := range o.cancels {
		if key == taskID || strings.HasPrefix(key, taskID+":") {
			cancel()
		}
	}
}

func (o *Orchestrator) registerCancel(key string, cancel context.CancelFunc) {
	o.mu.Lock()
	o.cancels[key] = cancel
	o.mu.Unlock()
}

func (o *Orchestrator) unregisterCancel(key string) {
	o.mu.Lock()
	delete(o.cancels, key)
	o.mu.Unlock()
}

func titleFromPrompt(prompt string) string {
	r := []rune(strings.TrimSpace(prompt))
	if len(r) == 0 {
		return "未命名森林"
	}
	if len(r) > 28 {
		return string(r[:28]) + "..."
	}
	return string(r)
}

func newID(prefix string) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(b)
}

func (o *Orchestrator) codexLogLine(line string) (string, bool) {
	level := normalizeLogLevel(o.store.GetSettings().LogLevel)
	line = strings.TrimSpace(line)
	if line == "" || level == LogLevelQuiet {
		return "", false
	}
	cleaned := strings.TrimSpace(strings.TrimPrefix(line, "stderr:"))
	if cleaned == "" {
		return "", false
	}
	lower := strings.ToLower(cleaned)
	if lower == "tokens used" || strings.HasPrefix(lower, "tokens used") || isMostlyNumber(cleaned) {
		return "", false
	}
	if level == LogLevelNormal && !importantLogLine(cleaned) {
		return "", false
	}
	return codex.Truncate(cleaned, 500), true
}

func importantLogLine(line string) bool {
	lower := strings.ToLower(line)
	keywords := []string{"error", "failed", "failure", "warning", "success", "created", "updated", "modified", "written", "wrote", "test", "build", "done", "complete", "完成", "失败", "错误", "警告", "通过", "生成", "写入", "修改", "创建", "测试", "构建"}
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return strings.HasPrefix(line, "##")
}

func isMostlyNumber(s string) bool {
	if s == "" {
		return false
	}
	digits := 0
	others := 0
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == ',' || r == '.' || r == ' ':
		default:
			others++
		}
	}
	return digits > 0 && others == 0
}

func userFacingMessage(s string) string {
	return strings.TrimSpace(s)
}

func getenvInt(key string, defaultValue int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultValue
	}
	return n
}

func getenvIntFallback(primaryKey, fallbackKey string, defaultValue int) int {
	if strings.TrimSpace(os.Getenv(primaryKey)) != "" {
		return getenvInt(primaryKey, defaultValue)
	}
	return getenvInt(fallbackKey, defaultValue)
}
