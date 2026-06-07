package app

import (
	"strings"
	"testing"
)

func TestIsProgressQuery(t *testing.T) {
	queries := []string{
		"进度怎么样？",
		"做到哪了",
		"还在运行吗",
		"any update?",
		"status?",
	}
	for _, q := range queries {
		if !isProgressQuery(q) {
			t.Fatalf("expected progress query: %q", q)
		}
	}
}

func TestIsProgressQueryDoesNotSwallowInstructions(t *testing.T) {
	instructions := []string{
		"修复状态管理 bug",
		"修改状态栏样式",
		"implement status page",
		"change task status display",
	}
	for _, q := range instructions {
		if isProgressQuery(q) {
			t.Fatalf("expected instruction, got progress query: %q", q)
		}
	}
}

func TestInferGoalStatus(t *testing.T) {
	status, _ := inferGoalStatus("Goal status: partial\n部分完成，还需要继续", nil)
	if status != "partial" {
		t.Fatalf("partial output status = %q", status)
	}
	status, _ = inferGoalStatus("任务量太大，不建议继续，需要用户拆分", nil)
	if status != "blocked" {
		t.Fatalf("blocked output status = %q", status)
	}
	status, _ = inferGoalStatus("Goal status: complete\n全部完成", nil)
	if status != "complete" {
		t.Fatalf("complete output status = %q", status)
	}
}

func TestLatestHumanProgressLimitsSnippets(t *testing.T) {
	long := strings.Repeat("x", maxProgressQuerySnippetRunes+50)
	task := &Task{
		GardenerProgress: []string{long},
		Trees: []*Tree{{
			Name:     long,
			Progress: []string{long},
		}},
	}

	got := latestHumanProgress(task)
	if strings.Contains(got, long) {
		t.Fatalf("progress query included unbounded text")
	}
	want := strings.Repeat("x", maxProgressQuerySnippetRunes) + "..."
	if strings.Count(got, want) < 2 {
		t.Fatalf("progress query did not include truncated snippets, got %q", got)
	}
}
