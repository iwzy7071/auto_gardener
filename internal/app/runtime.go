package app

import (
	"fmt"
	"strings"
	"time"
)

const (
	runtimeSeverityOK      = "ok"
	runtimeSeverityInfo    = "info"
	runtimeSeverityWarning = "warning"
	runtimeSeverityBlocked = "blocked"
)

func buildTaskRuntime(t *Task, now time.Time) *TaskRuntime {
	if t == nil {
		return nil
	}
	rt := &TaskRuntime{
		Phase:          taskPhase(t),
		Severity:       runtimeSeverityOK,
		CanAskProgress: t.Status == StatusRunning,
		CanResume:      t.Status == StatusFinished,
	}
	if !t.CreatedAt.IsZero() {
		rt.DurationSeconds = int64(now.Sub(t.CreatedAt).Seconds())
		if rt.DurationSeconds < 0 {
			rt.DurationSeconds = 0
		}
	}
	latest := latestTaskActivity(t)
	if latest != nil {
		at := *latest
		rt.LatestActivityAt = &at
		rt.IdleSeconds = int64(now.Sub(at).Seconds())
		if rt.IdleSeconds < 0 {
			rt.IdleSeconds = 0
		}
	}
	for _, tr := range t.Trees {
		if tr == nil {
			continue
		}
		rt.TotalTrees++
		if tr.Status == StatusFinished {
			rt.FinishedTrees++
		} else {
			rt.RunningTrees++
		}
	}
	rt.Cue, rt.Severity = runtimeCue(t, rt)
	return rt
}

func taskPhase(t *Task) string {
	if t == nil {
		return "unknown"
	}
	if t.Status == StatusFinished {
		if t.StopRequested {
			return "stopped"
		}
		return "finished"
	}
	if len(t.Trees) == 0 {
		return "planning"
	}
	for _, tr := range t.Trees {
		if tr != nil && tr.Status != StatusFinished {
			if tr.IsValidation {
				return "validating"
			}
			return "running_subtasks"
		}
	}
	if t.GardenerStatus == StatusRunning {
		return "deciding"
	}
	return "running"
}

func latestTaskActivity(t *Task) *time.Time {
	if t == nil {
		return nil
	}
	var latest *time.Time
	consider := func(v *time.Time) {
		if v == nil || v.IsZero() {
			return
		}
		if latest == nil || v.After(*latest) {
			cp := *v
			latest = &cp
		}
	}
	consider(t.LastProgressAt)
	if len(t.GardenerProgress) > 0 {
		consider(&t.UpdatedAt)
	}
	for _, tr := range t.Trees {
		if tr == nil {
			continue
		}
		consider(tr.UpdatedAtPtr())
		consider(tr.StartedAt)
		consider(tr.CompletedAt)
	}
	if latest == nil && !t.UpdatedAt.IsZero() {
		latest = &t.UpdatedAt
	}
	return latest
}

func (tr *Tree) UpdatedAtPtr() *time.Time {
	if tr == nil || tr.UpdatedAt.IsZero() {
		return nil
	}
	return &tr.UpdatedAt
}

func runtimeCue(t *Task, rt *TaskRuntime) (string, string) {
	if t == nil || rt == nil {
		return "", runtimeSeverityOK
	}
	if t.Status == StatusFinished {
		if t.StopRequested {
			return "任务已停止。如需继续，可以点击继续任务，Gardener 会先检查已有进度。", runtimeSeverityInfo
		}
		return "任务当前已结束。如结果不完整，可以点击继续任务。", runtimeSeverityOK
	}
	if rt.TotalTrees == 0 {
		return "Gardener 正在规划任务。查询进度不会中断正在运行的工作。", runtimeSeverityInfo
	}
	if rt.IdleSeconds >= int64(watchdogBlockedAfter().Seconds()) {
		return fmt.Sprintf("已经约 %d 分钟没有新的输出。底层 CLI 可能卡住、断网或等待模型响应；你可以询问进度，必要时点击继续任务让 Gardener 重新检查。", rt.IdleSeconds/60), runtimeSeverityBlocked
	}
	if rt.IdleSeconds >= int64(watchdogStaleAfter().Seconds()) {
		return fmt.Sprintf("已经约 %d 分钟没有新的输出，但任务仍标记为运行中。查询进度是安全的，不会打断任务。", rt.IdleSeconds/60), runtimeSeverityWarning
	}
	if rt.RunningTrees > 0 {
		return fmt.Sprintf("%d 个子任务正在运行，%d 个已完成。", rt.RunningTrees, rt.FinishedTrees), runtimeSeverityOK
	}
	return "子任务已返回，Gardener 正在验证或判断下一步。", runtimeSeverityInfo
}

func hasRecentWatchdogCue(t *Task, latest *time.Time) bool {
	if t == nil {
		return false
	}
	cutoff := time.Now().Add(-30 * time.Minute)
	if latest != nil && latest.After(cutoff) {
		cutoff = latest.Add(-2 * time.Second)
	}
	for i := len(t.Messages) - 1; i >= 0; i-- {
		msg := t.Messages[i]
		if msg.Role != RoleSystem {
			continue
		}
		if msg.CreatedAt.Before(cutoff) {
			return false
		}
		if strings.Contains(msg.Content, "任务状态提示") || strings.Contains(msg.Content, "没有新的输出") {
			return true
		}
	}
	return false
}
