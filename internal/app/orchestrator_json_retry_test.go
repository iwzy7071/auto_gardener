package app

import (
	"context"
	"strings"
	"sync"
	"testing"

	"auto_gardener/internal/codex"
)

type sequenceRunner struct {
	mu      sync.Mutex
	outputs []string
	prompts []string
}

func (r *sequenceRunner) Run(ctx context.Context, req codex.RunRequest) codex.RunResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prompts = append(r.prompts, req.Prompt)
	if len(r.outputs) == 0 {
		return codex.RunResult{Output: `{"message_to_user":"","forest_finished":true,"needs_clarification":false,"clarification_question":"","trees":[]}`}
	}
	out := r.outputs[0]
	r.outputs = r.outputs[1:]
	return codex.RunResult{Output: out}
}

func (r *sequenceRunner) Prompts() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.prompts...)
}

func TestGardenerPlanInvalidJSONAutoRetries(t *testing.T) {
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	newTestTask(t, store, "forest_plan_json_retry")
	runner := &sequenceRunner{outputs: []string{
		"我会先规划，但这不是 JSON。",
		`{"message_to_user":"","forest_finished":false,"needs_clarification":false,"clarification_question":"","trees":[{"name":"实现","objective":"完成修复","prompt":"完成修复并报告。","scope":["相关文件"]}]}`,
	}}
	orch := NewOrchestrator(store, runner, store.DataDir(), "")

	plan := orch.runGardenerPlan(context.Background(), "forest_plan_json_retry", "继续")
	if plan.ForestFinished || len(plan.Trees) != 1 {
		t.Fatalf("plan did not recover after retry: %+v", plan)
	}
	prompts := runner.Prompts()
	if len(prompts) != 2 {
		t.Fatalf("runner call count = %d, want 2", len(prompts))
	}
	if !strings.Contains(prompts[1], "自动重试一次") || !strings.Contains(prompts[1], "上一次无效输出摘要") {
		t.Fatalf("retry prompt missing repair guidance:\n%s", prompts[1])
	}
	got, ok := store.GetTask("forest_plan_json_retry")
	if !ok {
		t.Fatal("task missing")
	}
	for _, msg := range got.Messages {
		if strings.Contains(msg.Content, "规划结果格式异常") {
			t.Fatalf("auto-recovered retry should not append pause message: %+v", got.Messages)
		}
	}
	if !strings.Contains(strings.Join(got.GardenerProgress, "\n"), "自动重试第 1 次") {
		t.Fatalf("expected retry progress log, got %+v", got.GardenerProgress)
	}
}

func TestGardenerDecisionInvalidJSONAutoRetries(t *testing.T) {
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	newTestTask(t, store, "forest_decision_json_retry")
	runner := &sequenceRunner{outputs: []string{
		"not-json",
		`{"message_to_user":"","forest_finished":true,"needs_clarification":false,"clarification_question":"","trees":[]}`,
	}}
	orch := NewOrchestrator(store, runner, store.DataDir(), "")

	plan := orch.runGardenerDecision(context.Background(), "forest_decision_json_retry", 1)
	if !plan.ForestFinished || len(plan.Trees) != 0 {
		t.Fatalf("decision retry did not recover to finished plan: %+v", plan)
	}
	prompts := runner.Prompts()
	if len(prompts) != 2 {
		t.Fatalf("runner call count = %d, want 2", len(prompts))
	}
	if !strings.Contains(prompts[1], "上一次 Gardener 后续判断输出不是合法 JSON") {
		t.Fatalf("decision retry prompt missing repair guidance:\n%s", prompts[1])
	}
	got, ok := store.GetTask("forest_decision_json_retry")
	if !ok {
		t.Fatal("task missing")
	}
	for _, msg := range got.Messages {
		if strings.Contains(msg.Content, "后续判断结果格式异常") {
			t.Fatalf("auto-recovered retry should not append pause message: %+v", got.Messages)
		}
	}
}
