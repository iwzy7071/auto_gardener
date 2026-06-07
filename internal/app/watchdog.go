package app

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func watchdogInterval() time.Duration {
	return getenvDurationSeconds("AUTO_GARDENER_WATCHDOG_INTERVAL_SECONDS", 30*time.Second)
}

func watchdogStaleAfter() time.Duration {
	return getenvDurationSeconds("AUTO_GARDENER_WATCHDOG_STALE_SECONDS", 10*time.Minute)
}

func watchdogBlockedAfter() time.Duration {
	return getenvDurationSeconds("AUTO_GARDENER_WATCHDOG_BLOCKED_SECONDS", 30*time.Minute)
}

func getenvDurationSeconds(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return time.Duration(n) * time.Second
}

func (o *Orchestrator) StartWatchdog() {
	interval := watchdogInterval()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			o.RunWatchdogOnce(time.Now())
		}
	}()
}

func (o *Orchestrator) RunWatchdogOnce(now time.Time) {
	if o == nil || o.store == nil {
		return
	}
	for _, t := range o.store.ListTasks() {
		if t == nil || t.Status != StatusRunning {
			continue
		}
		rt := buildTaskRuntime(t, now)
		if rt == nil || rt.IdleSeconds < int64(watchdogStaleAfter().Seconds()) {
			continue
		}
		if hasRecentWatchdogCue(t, rt.LatestActivityAt) {
			continue
		}
		minutes := rt.IdleSeconds / 60
		if minutes < 1 {
			minutes = 1
		}
		cue := fmt.Sprintf("【任务状态提示】这个任务已经约 %d 分钟没有新的输出。底层 CLI 可能仍在运行、等待模型响应，或已经卡住。你可以直接询问进度；查询进度不会中断任务。如果后续仍无变化，可以点击“继续任务”，Gardener 会重新检查当前文件和报告后接着处理。", minutes)
		if rt.Severity == runtimeSeverityBlocked {
			cue = fmt.Sprintf("【任务状态提示】这个任务已经约 %d 分钟没有新的输出，可能已卡住或底层 CLI/模型连接异常。建议点击“继续任务”，Gardener 会先诊断已有进度，再决定继续、拆分或给出失败原因。", minutes)
		}
		o.appendSystemMessage(t.ID, cue)
		o.store.AppendGardenerLog(t.ID, "Watchdog 已提示用户：任务长时间无新输出。")
	}
}
