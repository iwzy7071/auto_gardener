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

const maxWatchdogDurationSeconds int64 = 7 * 24 * 60 * 60

func getenvDurationSeconds(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 || n > maxWatchdogDurationSeconds {
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
	for _, t := range o.store.ListRunningTasks() {
		if t == nil {
			continue
		}
		rt := buildTaskRuntime(t, now)
		if rt == nil || rt.IdleSeconds < int64(watchdogStaleAfter().Seconds()) {
			continue
		}
		if t.LastWatchdogAt != nil && now.Sub(*t.LastWatchdogAt) < watchdogStaleAfter() {
			continue
		}
		if hasRecentWatchdogCue(t, rt.LatestActivityAt) {
			continue
		}
		minutes := rt.IdleSeconds / 60
		if minutes < 1 {
			minutes = 1
		}
		mark := now
		_, _ = o.store.UpdateTask(t.ID, func(t *Task) {
			t.LastWatchdogAt = &mark
		})
		if o.hasTaskProcesses(t.ID) {
			o.store.AppendGardenerLog(t.ID, fmt.Sprintf("Watchdog 检测到任务约 %d 分钟无新输出，但底层 CLI 进程仍在运行；为避免主动中断，本轮仅记录并延后后台自查。", minutes))
			continue
		}
		o.store.AppendGardenerLog(t.ID, fmt.Sprintf("Watchdog 检测到任务约 %d 分钟无新输出，且未发现仍受 Gardener 管理的底层进程，已交由 Gardener 后台自查，不直接通知用户。", minutes))
		o.startForestRun(t.ID, buildWatchdogInstruction(t, minutes, rt.Severity))
	}
}
